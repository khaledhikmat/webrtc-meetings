const userName = "User-"+Math.floor(Math.random() * 100000)

let mediaConstraints = {
    audio: false, // We dont want an audio track
    video: true // ...and we want a video track
}

let peerConfiguration = {
    iceServers:[
        {
            urls:[
              'stun:stun.l.google.com:19302',
              'stun:stun1.l.google.com:19302'
            ]
        }
    ]
}

let peerConnection; //the peerConnection that the two clients use to talk
let localStream; //a var to hold the local video stream

const offer = async (meetingId, meetingName, type, elementId) => {
    await fetchUserMedia(elementId);
    document.getElementById(elementId).srcObject = localStream;

    await createPeerConnection(meetingId, meetingName, type);

    const offer = await peerConnection.createOffer();
    peerConnection.setLocalDescription(offer);

    localStream.getTracks().forEach(track => peerConnection.addTrack(track, localStream));
}

const fetchUserMedia = (elementId) => {
    return new Promise(async(resolve, reject)=>{
        try{
            const stream = await navigator.mediaDevices.getUserMedia(mediaConstraints);
            document.getElementById(elementId).srcObject = stream;
            localStream = stream
            resolve();    
        }catch(err){
            console.log(err);
            reject()
        }
    })
}

const createPeerConnection = (meetingId, meetingName, type)=>{
    return new Promise(async(resolve, reject)=>{
        //RTCPeerConnection is the thing that creates the connection
        //we can pass a config object, and that config object can contain stun servers
        //which will fetch us ICE candidates
        peerConnection = await new RTCPeerConnection(peerConfiguration)

        // localStream.getTracks().forEach(track=>{
        //     //add localtracks so that they can be sent once the connection is established
        //     peerConnection.addTrack(track,localStream);
        // })

        peerConnection.addEventListener("signalingstatechange", (event) => {
            console.log(event);
            console.log(peerConnection.signalingState)
        });

        peerConnection.addEventListener('icecandidate', async e=>{
            console.log('........Ice candidate found!......')
            console.log(e)
            if(e.candidate === null){
                // Send the offer request to the server
                let offerRequest = {
                    meetingId: meetingId,
                    name: meetingName,
                    Username: userName,
                    SD: peerConnection.localDescription
                }
                const {error, sd} = await fetch(type == "share" ? "/api/share" : "/api/participate", {
                    method: "POST",
                    headers: {
                        "Content-Type": "application/json"
                    },
                    body: JSON .stringify(offerRequest) 
                }).then(r => r.json())

                // Assign the returned session description as remote
                console.log("error", error) 
                console.log("answer", sd) 
                if (!error) {
                    await peerConnection.setRemoteDescription(new RTCSessionDescription(sd))
                }
            }
        })
        
        peerConnection.addEventListener('track',e=>{
            console.log("Got a track from the other peer!! How excting")
            console.log(e)
            e.streams[0].getTracks().forEach(track=>{
                console.log("Here's an exciting moment... fingers cross")
            })
        })
        resolve();
    })
}
