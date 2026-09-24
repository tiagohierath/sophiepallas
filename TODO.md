# TODO

sophie-nectar. Philosophy and full usage in SOPHIE-NECTAR.md.

## Goal
5 to 10 minute tracks, EDM and simple IDM ambient, built from found samples.
Strudel locally, sections as files, ffmpeg as the tape machine. No DAW.

## Done
- [x] Strudel headless from the command line (wirbel + chromium via nix)
- [x] Local samples served from samples/ during the render
- [x] One file per section, `// cycles:` or `// duration:` header
- [x] prelude.strudel: one tempo and one samples() for the whole song
- [x] `bin/render`, `bin/stitch` (optional crossfade), `bin/song`
- [x] `bin/look`: levels table, waveform and spectrogram pictures, inline in kitty
- [x] `bin/sample`: trim, normalize and file any found audio
- [x] `bin/shell`: sox and rubberband alongside ffmpeg and mpv
- [x] Console: live refresh, stale marks, cursor, inline waveforms, playhead, scrolling
- [x] git, and a page in the obsidian vault

## Next
- [ ] Write a real 5 to 10 minute track and see what the workflow is missing
- [ ] Feed samples/ with real found sounds
- [ ] Decide how layering works when a section needs more than one stack
- [ ] Maybe: per section gain or fade in arrangement.txt
- [ ] Maybe: a master step, loudnorm to -14 LUFS, on the way to mp3
- [ ] Try dough (dough.strudel.cc) as a browserless second render backend
