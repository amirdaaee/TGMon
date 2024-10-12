package ffmpeg

import (
	"fmt"

	"github.com/google/uuid"
)

func GenThumnail(container *FFmpegContainer, url string, at int) ([]byte, error) {
	fname := fmt.Sprintf("/config/%s.jpg", uuid.NewString())
	cmd := []string{"ffmpeg", "-ss", fmt.Sprintf("%d", at), "-i", url, "-frames:v", "1", fname}
	if res, err := container.Exec(cmd); err != nil {
		return nil, fmt.Errorf("error execution: %s - command: %+v", err, cmd)
	} else if res != 0 {
		return nil, fmt.Errorf("ffmpeg exited non-zero (%d) - command: %+v", res, cmd)
	}
	return container.Open(fname)
}
