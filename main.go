package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"
)

func usage() {
	fmt.Fprintf(os.Stderr, "usage: chapter-thumbnails <video> <dir>\n")
	flag.PrintDefaults()
	os.Exit(2)
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	var (
		formatF = flag.String("f", "png", "Thumbnail format")
	)
	flag.Usage = usage
	flag.Parse()
	args := flag.Args()
	if len(args) != 2 {
		usage()
	}
	videoPath, outDir := args[0], args[1]

	chapters, err := VideoChapters(videoPath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return err
	}
	var eg errgroup.Group
	for _, c := range chapters {
		c := c
		eg.Go(func() error {
			thumbPath := filepath.Join(outDir, fmt.Sprintf("%s.%s", c.Title, *formatF))
			return CreateThumbnail(videoPath, thumbPath, c.Start)
		})
	}
	return eg.Wait()
}

func VideoChapters(videoPath string) ([]*Chapter, error) {
	tmpFile, err := ioutil.TempFile("", "chapter-thumbnails-*.txt")
	if err != nil {
		return nil, err
	}
	defer tmpFile.Close()
	defer os.Remove(tmpFile.Name())
	buf := &bytes.Buffer{}
	cmd := exec.Command(
		"ffmpeg",
		"-i", videoPath,
		"-y",
		"-f", "ffmetadata", tmpFile.Name(),
	)
	cmd.Stdout = buf
	cmd.Stderr = buf
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ffmpeg error: %q\n%s: %s", cmd.String(), buf, err)
	}
	return ParseChapters(tmpFile)
}

func CreateThumbnail(videoPath, thumbPath string, offset time.Duration) error {
	buf := &bytes.Buffer{}

	cmd := exec.Command(
		"ffmpeg",
		"-ss", fmt.Sprintf("%f", offset.Seconds()),
		"-i", videoPath,
		"-y",
		"-frames:v", "1",
		thumbPath,
	)
	cmd.Stdout = buf
	cmd.Stderr = buf
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg error: %q\n%s: %s", cmd.String(), buf, err)
	}
	return nil
}

type Chapter struct {
	Title string
	Start time.Duration

	timebase struct {
		numerator   int64
		denominator int64
	}
	start int64
}

func (s *Chapter) update() {
	if s.Start == 0 &&
		s.timebase.numerator != 0 &&
		s.timebase.denominator != 0 &&
		s.start != 0 {
		s.Start = time.Duration(float64(s.start) * (float64(s.timebase.numerator) / float64(s.timebase.denominator)) * float64(time.Second))
	}
}

func ParseChapters(r io.Reader) ([]*Chapter, error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(data), "\n")
	var chapter *Chapter
	var chapters []*Chapter
	for _, line := range lines {
		if line == "[CHAPTER]" {
			chapter = &Chapter{}
			chapters = append(chapters, chapter)
			continue
		} else if chapter == nil {
			continue
		}

		kv := strings.Split(line, "=")
		if len(kv) != 2 {
			continue
		}
		k, v := kv[0], kv[1]
		k = strings.ToLower(k)
		switch k {
		case "title":
			chapter.Title = v
		case "start":
			chapter.start, err = strconv.ParseInt(v, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("bad start: %q", line)
			}
		case "timebase":
			nd := strings.Split(v, "/")
			if len(nd) != 2 {
				return nil, fmt.Errorf("bad timebase: %q", line)
			}
			chapter.timebase.numerator, err = strconv.ParseInt(nd[0], 10, 64)
			if err != nil {
				return nil, fmt.Errorf("bad timebase: %q", line)
			}
			chapter.timebase.denominator, err = strconv.ParseInt(nd[1], 10, 64)
			if err != nil {
				return nil, fmt.Errorf("bad timebase: %q", line)
			}
		}
		chapter.update()
	}

	return chapters, nil
}
