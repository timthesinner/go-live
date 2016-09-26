# TL&DR
This project is an experiment to consume an over the air broadcast(s) ultimately producing a live stream viewable in a modern web browser.  The ultimate goal of this project is to leverage existing tools and utilities, not rewrite or patch them.

# Requirements
- Consume an OTA Television broadcast.
- Consume an OTA Radio broadcast.
- Produce a live stream combining the TV video and Radio audio sources.
- Encoder must encode at line speed.
- Encoded stream bit-rate must not exceed 2,500 kb/s.

# Tools
- [FFMPEG](https://ffmpeg.org) to encode streams.
- [WinTV](http://www.hauppauge.com/site/support/support_wintv7.html) native Windows support for a Hauppauge TV USB tuner.
- [GoLang](https://golang.org/project/) nothing beats a statically compiled executable for gluing services together.

# Learnings
As I worked through issues, I tracked some key learnings as I encountered them.

### Limited Linux Support
I initially tried to stitch together this solution using Linux.  I ran into major problems with the drivers that support my USB tuner though.  I simply could not get Linux to tune, and hold open a HD channel.  This forced me to transition to the Windows platform to leverage WinTV's formal support for my tuner.  If you can get your tuner card working on Linux, you can use dshow with FFMPEG to avoid the need for a large MPEG-2 TS file and wont need to use the piper executable.

### Limited FFMPEG streaming support
FFMPEG does not do a very good job of pulling in a live source from the filesystem.  FFMPEG reads the file size when it starts and runs until it digests that amount of data.  This works great for a file that is already baked, but does not work at all for a file that grows until recording stops.  I wrote the piper utility to solve this problem for me, it simply reads a file continuously, when it encounters new data it writes it to stdout.

### Modern Browser support for Streamable content
Modern Browsers do not seem to update their file size bound when streaming content using range requests.  I expected a native implementation to update the upper bound if the range request's response headers indicated that the file size had grown.  Unfortunately this is not the case.  I wrote the streamer utility to facilitate a stream of a webm source file that grows until recording stops.
