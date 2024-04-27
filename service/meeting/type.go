package meeting

type IService interface {
	GetActiveMeetings() ([]Meeting, error)
	GetMeetingById(id string) (Meeting, error)
	NewMeeting(n string) (Meeting, error)
	UpdateMeeting(u Meeting) error
	Finalize()
}
