## TODO: use os.Exec. its much better cuz we can actually kill processes.
## TODO: kill the ffmpeg process on ctrl+c. (if terminal window terminates it'll probably leave a zombie process)
## TODO: calculate total progress based on file sizes instead of counts of files for a more accurate progress percentage.
## TODO: customizable encoding settings, using tui elements to control and select them. (not gonna work on this in the near future)
## TODO: ask user if they want the old videos deleted. (probably not gonna work on this in the near future too)
## FIXME: tui is stuck forever if file is already encoded.
## TODO: when passing in a directory, list all video files in that directory and have the user choose which ones they want encoded
