package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pion/webrtc/v4"
)

var (
	stream chan interface{}
	//TODO: This is totally wrong! We need to allow multiple meetings at the same time
	localTrack *webrtc.TrackLocalStaticRTP
)

var peerConnectionConfig = webrtc.Configuration{
	ICEServers: []webrtc.ICEServer{
		{
			URLs: []string{
				"stun:stun.l.google.com:19302",
				"stun:stun1.l.google.com:19302",
			},
		},
	},
}

type offerRequest struct {
	MeetingID string                    `json:"meetingId"`
	Name      string                    `json:"name"`
	Username  string                    `json:"username"`
	SD        webrtc.SessionDescription `json:"sd"`
}

type offerResponse struct {
	Error string                    `json:"error"`
	SD    webrtc.SessionDescription `json:"sd"`
}

func homeRoutes(canxCtx context.Context, r *gin.Engine, api *webrtc.API) {
	// Capture results from different meetings
	stream = captureResults(canxCtx)

	r.GET("/", func(c *gin.Context) {
		target := "index.html"

		meetings, err := MeetingService.GetActiveMeetings()
		if err != nil {
			c.HTML(200, target, gin.H{
				"Tab":      "Home",
				"Error":    fmt.Sprintf("meetings returned an error: %s", err.Error()),
				"Meetings": meetings,
			})
			return
		}

		c.HTML(200, target, gin.H{
			"Tab":      "Home",
			"Error":    "",
			"Meetings": meetings,
		})
	})

	r.GET("/meeting", func(c *gin.Context) {
		target := "sharer.html"
		if c.Query("id") == "" {
			c.HTML(200, target, gin.H{
				"Tab":   "Home",
				"Error": "Meeting id is missing!",
			})
			return
		}

		if c.Query("t") == "participant" {
			target = "participant.html"
		}

		meeting, err := MeetingService.GetMeetingById(c.Query("id"))
		if err != nil {
			c.HTML(200, target, gin.H{
				"Tab":   "Home",
				"Error": "Meeting is missing!",
			})
			return
		}

		c.HTML(200, target, gin.H{
			"Tab":     "Home",
			"Error":   "",
			"Meeting": meeting,
		})
	})

	r.POST("/meetings", func(c *gin.Context) {
		target := "index.html"

		if c.Query("name") == "" {
			c.HTML(200, target, gin.H{
				"Tab":   "Home",
				"Error": "Meeting name is missing!",
			})
			return
		}

		_, err := MeetingService.NewMeeting(c.Query("name"))
		if err != nil {
			c.HTML(200, target, gin.H{
				"Tab":   "Home",
				"Error": fmt.Sprintf("new meeting returned an error: %s", err.Error()),
			})
			return
		}

		meetings, err := MeetingService.GetActiveMeetings()
		if err != nil {
			c.HTML(200, target, gin.H{
				"Tab":      "Home",
				"Error":    fmt.Sprintf("meetings returned an error: %s", err.Error()),
				"Meetings": meetings,
			})
			return
		}

		c.HTML(200, target, gin.H{
			"Tab":      "Home",
			"Error":    "",
			"Meetings": meetings,
		})
	})

	//=========================
	// APIs
	//=========================
	r.POST("/api/share", func(c *gin.Context) {

		var offer offerRequest
		if err := c.ShouldBindJSON(&offer); err != nil {
			c.JSON(500, offerResponse{
				Error: fmt.Sprintf("unable to parse offer from sharer: %s", err.Error()),
				SD:    webrtc.SessionDescription{},
			})
			return
		}

		meeting, err := MeetingService.GetMeetingById(offer.MeetingID)
		if err != nil {
			c.JSON(500, offerResponse{
				Error: fmt.Sprintf("unable to get a meeting: %s", err.Error()),
				SD:    webrtc.SessionDescription{},
			})
			return
		}

		// Establish a sharer
		answer, err := establishPeer(canxCtx, meeting.ID, api, offer, true, stream)
		if err != nil {
			c.JSON(500, offerResponse{
				Error: fmt.Sprintf("unable to start a sharer: %s", err.Error()),
				SD:    webrtc.SessionDescription{},
			})
			return
		}

		c.JSON(200, offerResponse{
			Error: "",
			SD:    answer,
		})
	})

	// New participant session
	r.POST("/api/participate", func(c *gin.Context) {

		var offer offerRequest
		if err := c.ShouldBindJSON(&offer); err != nil {
			c.JSON(500, offerResponse{
				Error: fmt.Sprintf("unable to parse offer from participant: %s", err.Error()),
				SD:    webrtc.SessionDescription{},
			})
			return
		}

		meeting, err := MeetingService.GetMeetingById(offer.MeetingID)
		if err != nil {
			c.JSON(500, offerResponse{
				Error: fmt.Sprintf("unable to get a meeting: %s", err.Error()),
				SD:    webrtc.SessionDescription{},
			})
			return
		}

		// Establish a participant for a meeting
		answer, err := establishPeer(canxCtx, meeting.ID, api, offer, false, stream)
		if err != nil {
			c.JSON(500, offerResponse{
				Error: fmt.Sprintf("unable to start a participant: %s", err.Error()),
				SD:    webrtc.SessionDescription{},
			})
			return
		}

		c.JSON(200, offerResponse{
			Error: "",
			SD:    answer,
		})
	})
}

func establishPeer(_ context.Context, meetingID string, api *webrtc.API, offer offerRequest, isSharer bool, resultsStream chan interface{}) (webrtc.SessionDescription, error) {
	fmt.Printf("meeting %s is starting - sharer: %t\n", meetingID, isSharer)

	// Create a new WBERTC peer connection
	// we (server) are always the recipient of an offer ....we respond with an answer
	peerConnection, err := api.NewPeerConnection(peerConnectionConfig)
	if err != nil {
		return webrtc.SessionDescription{}, err
	}

	if !isSharer {
		fmt.Println("***** participant mode")
		// Do not allow particpants until the localTrack is established
		// TODO: Of course this is wrong!
		if localTrack == nil {
			return webrtc.SessionDescription{}, fmt.Errorf("sharer has not started yet")
		}

		rtpSender, err := peerConnection.AddTrack(localTrack)
		if err != nil {
			return webrtc.SessionDescription{}, err
		}

		// Read incoming RTCP packets
		// Before these packets are returned they are processed by interceptors. For things
		// like NACK this needs to be called.
		go func() {
			rtcpBuf := make([]byte, 1500)
			for {
				if _, _, rtcpErr := rtpSender.Read(rtcpBuf); rtcpErr != nil {
					return
				}
			}
		}()
	} else {
		fmt.Println("***** sharer mode")
		// Allow us to receive 1 video track
		if _, err = peerConnection.AddTransceiverFromKind(webrtc.RTPCodecTypeVideo); err != nil {
			return webrtc.SessionDescription{}, err
		}

		peerConnection.OnTrack(func(remoteTrack *webrtc.TrackRemote, _ *webrtc.RTPReceiver) {
			fmt.Printf("Track acquired - kind: %v - codec: %v\n", remoteTrack.Kind(), remoteTrack.Codec().MimeType)

			// Create a local track, all our SFU clients will be fed via this track
			lclTrack, localTrackErr := webrtc.NewTrackLocalStaticRTP(remoteTrack.Codec().RTPCodecCapability, "video", "pion")
			if localTrackErr != nil {
				panic(localTrackErr)
			}

			localTrack = lclTrack

			rtpBuf := make([]byte, 1400)
			for {
				i, _, readErr := remoteTrack.Read(rtpBuf)
				if readErr != nil {
					panic(readErr)
				}

				// ErrClosedPipe means we don't have any subscribers, this is ok if no peers have connected yet
				if _, err = localTrack.Write(rtpBuf[:i]); err != nil && !errors.Is(err, io.ErrClosedPipe) {
					panic(err)
				}
			}
		})
	}

	// Set the handler for ICE connection state
	// This will notify you when the peer has connected/disconnected
	peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		fmt.Printf("Connection State has changed to: %s - sharer: %t\n", connectionState.String(), isSharer)

		if connectionState == webrtc.ICEConnectionStateConnected {
			fmt.Println("ICE state connected")
		} else if connectionState == webrtc.ICEConnectionStateFailed {
			fmt.Println("ICE connection closed or failed")
			resultsStream <- "closing due to error"
			return
		} else if connectionState == webrtc.ICEConnectionStateClosed {
			fmt.Println("ICE connection closedd")
			resultsStream <- "closing normally"
			return
		}
	})

	// fmt.Printf("remote desc %v", offer.SD)
	err = peerConnection.SetRemoteDescription(offer.SD)
	if err != nil {
		return webrtc.SessionDescription{}, err
	}

	// Create answer
	answer, err := peerConnection.CreateAnswer(nil)
	if err != nil {
		return webrtc.SessionDescription{}, err
	}

	// Sets the LocalDescription, and starts our UDP listeners
	err = peerConnection.SetLocalDescription(answer)
	if err != nil {
		return webrtc.SessionDescription{}, err
	}

	return answer, nil
}

func captureResults(canxCtx context.Context) chan interface{} {
	stream := make(chan interface{}, 10)

	go func() {
		defer close(stream)

		for {
			select {
			case <-canxCtx.Done():
				fmt.Println("results stream cancelled")
				// Wait until downstream processors are done
				fmt.Println("wait for 2 seconds until downstream processors are cancelled...")
				time.Sleep(2 * time.Second)
				return
			case r := <-stream:
				fmt.Printf("received a result %v\n", r)
				if errorResult, ok := r.(error); ok {
					fmt.Printf("received an error %v\n", errorResult)
				}
			}
		}
	}()

	return stream
}
