- [x] query ffmpeg binary for available encoders and list the ones we know
- [-] when passing in a directory, list all video files in that directory and have the user choose which ones they want encoded
    - [x] Scrollable viewport for the files. They currently get cut if there's too many.
- [ ] CLI Option/Flag to choose the output directory for the encoded video(s)
- [ ] Show FPS, bitrate, and current file size when encoding.
    - Has to be parsed from the ffmpeg output
- [ ] calculate total progress based on file sizes instead of counts of files for a more accurate progress percentage.
    might not do this.
