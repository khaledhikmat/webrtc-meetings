package meeting

import (
	"fmt"

	"github.com/google/uuid"
)

// TODO: in-memory
var meetings = []Meeting{
	{
		ID:           "1000",
		Name:         "meeting1",
		Duration:     1000,
		Participants: 0,
		Active:       true,
	},
	{
		ID:           "1001",
		Name:         "meeting2",
		Duration:     1230,
		Participants: 0,
		Active:       true,
	},
}

type meetingService struct {
}

func NewService() IService {
	return &meetingService{}
}

func (m *meetingService) GetActiveMeetings() ([]Meeting, error) {

	return meetings, nil
}

func (m *meetingService) GetMeetingById(id string) (Meeting, error) {
	for _, meeting := range meetings {
		if meeting.ID == id {
			return meeting, nil
		}
	}

	return Meeting{}, fmt.Errorf("unable to find a meeting by id %s", id)
}

func (m *meetingService) NewMeeting(name string) (Meeting, error) {
	meeting := Meeting{
		ID:           uuid.NewString(),
		Name:         name,
		Active:       true,
		Participants: 0,
		Duration:     0,
	}

	// TODO: Wrong
	meetings = append(meetings, meeting)
	return meeting, nil
}

func (m *meetingService) UpdateMeeting(u Meeting) error {
	_, err := m.GetMeetingById(u.ID)
	if err != nil {
		return err
	}

	// TODO: Wrong

	// Remove current meeting
	var idx = 0
	var meeting Meeting
	for idx, meeting = range meetings {
		if meeting.ID == u.ID {
			break
		}
	}

	meetings = append(meetings[:idx], meetings[idx+1:]...)

	// Re-add it
	meetings = append(meetings, u)
	return nil
}

func (m *meetingService) Finalize() {
}
