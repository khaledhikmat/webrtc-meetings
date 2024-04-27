This project is about creating online meetings using WebRTC. There are two meeting types:
- Presentation where a presenter shared screen and participants watch, listen and ask questions via a data channel.
- Conversation where two participants exchange video, audio and data channel.

```bash
go mod init github.com/khaledhikmat/webrtc-meetings
go get -u github.com/gin-gonic/gin
go get -u github.com/gin-contrib/cors
go get -u github.com/joho/godotenv
go get -u github.com/pion/webrtc/v4
go get -u github.com/google/uuid
```
