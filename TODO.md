# TODO

sophie-nectar. Philosophy and full usage in SOPHIE-NECTAR.md.

## Goal
Make long songs with Strudel, running locally (no strudel.cc dependency), with my own
samples, built from several sections, stitched together with ffmpeg only. No DAW.

## Done
- [x] Run Strudel locally, headless, from the command line (wirbel + chromium via nix)
- [x] Load local samples (@strudel/sampler serves samples/ during the render)
- [x] One file per section, `// cycles:` or `// duration:` header
- [x] `bin/render <song>` renders every section to wav, faster than real time
- [x] `bin/stitch <song> [xfade]` concatenates by arrangement.txt, optional crossfade
- [x] `bin/song <song>` renders, stitches, plays in mpv
- [x] `bin/nectar` panel: song info, notes, sections, [r] render [p] play
- [x] demo song proving the whole chain (1:52 out of 3 sections, 5 slots)
- [x] Panel fills the terminal, one mpv at a time, playback dies on quit and on ctrl-c

## Goal shape
5 to 10 minute tracks, edm and simple idm ambient, built from found samples.
That means many sections at one shared tempo, and a quick way to intake random audio.

## Doing
- [ ] prelude.strudel per song: tempo, samples() and helpers shared by every section
- [ ] bin/sample: drop any audio file in, get a trimmed normalized wav in samples/<name>/
- [ ] Panel scrolls the section list, a 10 section song has to fit
- [ ] Best possible panel beside Helix:
      - live, redraws by itself when a .strudel changes, no keypress needed
      - stale markers: section edited after its last render
      - cursor over sections, render/play/look just the one under it
      - inline waveform per section, drawn in blocks
      - playhead over the arrangement while mpv plays (mpv ipc socket)
      - status line saying what just happened and how long it took
- [ ] bin/look: waveform + spectrogram PNGs, peak/rms per section, inline in kitty
- [ ] Panel shows peak dB per section and flags silent or clipped ones, [l] runs look
- [ ] bin/shell: nix shell with sox, rubberband, ffmpeg, mpv on PATH
- [ ] sox and rubberband recipes in the doc
- [ ] git init and first commit

## Next
- [ ] Put real samples in samples/ and write a real song
- [ ] Decide how mixing works when sections need layers (ffmpeg amix vs one strudel stack)
- [ ] Try dough (dough.strudel.cc) as a browserless second render backend
- [ ] Maybe: per section gain or fade in the arrangement file
