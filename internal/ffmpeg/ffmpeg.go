package ffmpeg

import (
	"fmt"
	"time"
)

func GenThumnail(container *FFmpegContainer, url string, at int) ([]byte, error) {
	fname := fmt.Sprintf("/config/%d.jpg", time.Now().Unix())
	cmd := []string{"ffmpeg", "-ss", fmt.Sprintf("%d", at), "-i", url, "-frames:v", "1", fname}
	if res, err := container.Exec(cmd); err != nil {
		return nil, fmt.Errorf("error execution: %s", err)
	} else if res != 0 {
		return nil, fmt.Errorf("ffmpeg exited non-zero (%d)", res)
	}
	return container.Open(fname)
}
