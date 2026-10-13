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

## The working loop this has to serve
Write a loop, save, hear it. Edit it, save it as another section. Edit more, save that
as the intro and the outro. The song grows by forking whatever is on screen.

- In Helix that fork is `:w sections/03-outro.strudel`, nothing needed from us
- [ ] Console watch mode: on save, re-render the section under the cursor and loop it
- [ ] Console can append a section to arrangement.txt, the organize half of the console

## Exploring: a browser UI
Open question Tiago is chewing on: strudel is declarative (write a pattern, it loops),
but he also wants to play layers imperatively (dough-style keyboard) and record them.
Resolution so far: imperative is the input method, declarative text is the storage. A
recorded take freezes into a strudel line (quantized to the song grid) and becomes a
section like any other. The UI is a disposable instrument, text files stay the truth.

- [x] Prototype the full UI, look only, no backend, served on localhost for Firefox,
      showing all modules at once (now replaced by the hub, proto/index.html)
- [x] Hub of existing OSS tools: strasbeat, dough keyboard (local), web-synth, mono82 (local)
- [x] dough keyboard: all letter + number keys, instruments sidebar (save/load patches)
- [ ] dough bug: clicks don't always register
- [ ] dough bug: attack behaves wrong a lot
- [ ] Infinite canvas (existing pan/zoom + drag libs) holding the local tools as panels:
      dough, mono82, strudel REPL. Remote tools stay as links (can't embed in an isolated page)
- [ ] First wire: mono82 steps trigger dough, over a same-origin message bus
- [x] RACK, one screen, no tab switching: proto/rack.html stacks the real local tools
      (dough keyboard, mono82) as rack units in same-origin iframes. Found tools only,
      nothing redrawn by hand. Run: python3 proto/serve.py, open localhost:8099/rack.html
- [ ] ONE screen to make good EDM ambient sounds: every layer plays together in time,
      hands-on controls, and the result is strudel text that becomes a section
- [ ] Vendor strasbeat (AGPL) + web-synth (MIT) locally so they can be rack units too
- [ ] Decide: move the console (tui/main.go) to Rust or keep Go. Tiago is learning Rust
      and knows Go. Suggestion: instruments stay found browser tools, Rust for the console

## Next
- [ ] Write a real 5 to 10 minute track and see what the workflow is missing
- [ ] Feed samples/ with real found sounds
- [ ] Decide how layering works when a section needs more than one stack
- [ ] Maybe: per section gain or fade in arrangement.txt
- [ ] Maybe: a master step, loudnorm to -14 LUFS, on the way to mp3
- [ ] Try dough (dough.strudel.cc) as a browserless second render backend
