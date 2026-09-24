# sophie-nectar

A music workstation made of text files, a renderer, and ffmpeg. No DAW, no session
file, no GUI. Command line only.

Built for 5 to 10 minute tracks: EDM and simple IDM ambient, made of found samples.

```
HELIX                    |  CONSOLE
make the music           |  understand it, organize it, listen to it,
                         |  render it, remember it
```

## Philosophy

A song is a directory of small deterministic programs plus an order to play them in.

1. **The source is text.** Every sound comes from a `.strudel` file I can read, diff,
   and version. Nothing lives inside a binary project file or a plugin preset.
2. **Sections are files, not a timeline.** A song is not one long recording being
   scrubbed through. It is `01-intro`, `02-main`, `03-outro`, each rendered on its own.
3. **Rendering is reproducible.** Same file in, same wav out, every time. Renders are
   throwaway artifacts. Delete `render/` and the song is still intact.
4. **ffmpeg is the tape machine.** Stitching, fading, mixing, trimming, normalizing,
   speed, pan, silence, format. Every edit is a command, so it can be repeated.
5. **The arrangement is a list.** `arrangement.txt` names sections in order and can
   repeat them. Changing the shape of a ten minute song is a few lines of text.
6. **One tempo per song.** `prelude.strudel` holds it, every section inherits it.
7. **Nothing is installed globally.** Toolchains come from nix on demand, node packages
   live in `.node/`. The project is the environment.
8. **Nothing is precious except the source.** `.strudel` files and `samples/` are the
   work. Everything else can be regenerated.

## Shape

```
section.strudel  ->  wirbel (headless render)  ->  section.wav  ->  ffmpeg  ->  final.wav
```

```
sophie-nectar/
├── SOPHIE-NECTAR.md        this file
├── TODO.md
├── bin/
│   ├── render              sections -> wav
│   ├── stitch              wav + arrangement -> final.wav
│   ├── look                waveforms, spectrograms, levels
│   ├── sample              any audio file -> a usable sample
│   ├── song                render, stitch, play
│   ├── nectar              the console (built from tui/)
│   ├── shell               a shell with sox, rubberband, ffmpeg, mpv
│   └── build               rebuild the console
├── tui/main.go             the console, one Go file, no dependencies
├── samples/                my own sounds, one folder per sound name
│   └── blip/blip.wav
└── songs/
    └── demo/
        ├── prelude.strudel     tempo and samples, shared by every section
        ├── sections/
        │   ├── 01-intro.strudel
        │   ├── 02-main.strudel
        │   └── 03-outro.strudel
        ├── arrangement.txt     play order, repeats allowed
        ├── notes.md            free text, shown in the console
        ├── render/             generated, disposable
        └── look/               pictures, disposable
```

## Commands

```bash
bin/nectar demo              # the console
bin/render demo              # render every section
bin/render demo main         # only sections whose name contains "main"
bin/stitch demo              # butt joint concat, sample accurate
bin/stitch demo 1.5          # concat with 1.5s crossfades
bin/look demo                # levels table, pictures, drawn inline in kitty
bin/sample ~/Downloads/x.wav # trim, normalize, file it under samples/
bin/song demo                # render, stitch, play
```

## The console

Helix on the left, `bin/nectar` on the right. The console never edits anything. It
reads the song off disk and runs the same scripts you would run by hand.

```
┌────────────────────────────────────────────────────────────────────┐
│ SONG  demo                                                         │
│ 3 sections  5 slots                                        = 01:52 │
│ cps 0.5                                                    120 bpm │
│                                                                    │
│ ━━━━━━━━━━━━━━━━━━━━━━━━ ─────────────────────────────────────────  │
│ playing 00:41                                              01:52   │
│                                                                    │
│ SECTIONS                                                           │
│ ────────────────────────────────────────────────────────────────── │
│ › 01-intro ▅▄▄▅▄▄▅▄▄▅▄▄▅▄▄▅▄▄▅▄▄▅▄▄                  ● -15.1 00:16 │
│   02-main  ▇▇▇▇▇▇▇▇▇▇▇▇▇▇▇▇▇▇▇▇▇▇▇▇                    -14.7 00:32 │
│   03-outro ▆▆▆▆▇▇▆▆▆▇▇▇▆▆▆▇▇▇▆▆▆▇▇█                    -16.6 00:16 │
│                                                                    │
│ ARRANGEMENT                                                        │
│ ────────────────────────────────────────────────────────────────── │
│ 01-intro > 02-main > 03-outro > 02-main > 03-outro                 │
│                                                                    │
│ NOTES                                                              │
│ ────────────────────────────────────────────────────────────────── │
│ intro: birds + drone                                               │
│                                                                    │
│ ● needs rendering   render in 24.1s                                │
│ j k move   enter hear   R render   l look                          │
│ r render all   p play   space stop   q quit                        │
└────────────────────────────────────────────────────────────────────┘
```

It refreshes itself. Save a section in Helix and within a moment the console marks it
`●`, meaning the file moved after its last render. `=` next to the song length means
every slot is backed by a real wav, `~` means part of it is still an estimate from
cycle counts.

| key | what it does |
| --- | --- |
| `j` `k` `g` `G` | move the cursor through the sections |
| `enter` | play the section under the cursor |
| `R` | render only the section under the cursor |
| `l` | look at the section under the cursor |
| `r` | render the whole song, then stitch |
| `s` | stitch only |
| `L` | look at the whole song |
| `p` | play the stitched song |
| `space` | stop playing |
| `q` | quit |

The waveform beside each section is its own envelope, all sections drawn to one shared
scale, so a quiet intro looks quiet next to a loud middle. The number after it is peak
dBFS, with `SILENT` or `CLIP` when something is wrong. While mpv plays, a playhead
rides across the top. Long songs scroll the section list around the cursor.

One Go file in `tui/`, no dependencies, rebuilt with `bin/build`.

## Conventions

**Tempo and samples** go in `songs/<song>/prelude.strudel`, which is glued on top of
every section before rendering:

```js
samples('http://localhost:5432')
setcps(0.5)
```

`cps` is cycles per second. At `0.5` a cycle is two seconds, which reads as 120 bpm in
four four. One tempo for the whole song is what keeps ten sections stitchable.

**Section length** goes in a comment at the top of the section, in cycles or seconds:

```js
// cycles: 32
// duration: 45
```

Default is 16 cycles. For a 5 to 10 minute track, 8 to 12 sections of 30 to 60 seconds
is a comfortable shape.

**Arrangement** is one section name per line, `#` for comments, repeats welcome:

```
01-intro
02-main
03-break
02-main
04-outro
```

With no `arrangement.txt`, sections play in filename order, which is why they are
numbered.

**Samples** are folders under `samples/`, one folder per sound name. `bin/render` serves
that directory on port 5432 while it renders. `bin/sample` is the way in:

```bash
bin/sample ~/Downloads/some\ field\ recording.wav birds
```

It lowercases the name, trims the silence off the front, peak normalizes to -1 dB,
converts to 44.1k wav, and files it as `samples/birds/birds-00.wav`. Drop more in with
the same name and they stack, so `n("<0 1 2>").s("birds")` picks between them.

## Gotchas

- **Use single quotes for `samples('...')`.** Strudel turns double quoted strings into
  mini notation, so `samples("http://...")` dies on the `/` with a parse error.
- `.strudel` files are executable JavaScript. Only render files you wrote or trust.
- Renders run faster than real time, but a ten minute song is still a minute or two of
  chromium work. Render one section at a time with `R` while writing.
- Crossfading shortens the song by the fade time at every join.
- **`.room()` does nothing in an offline render.** Two sections identical except for
  `.room(0.6)` come out byte for byte the same. The reverb impulse is built
  asynchronously and the render finishes first. `.delay()` does work. Reverb belongs to
  the tape machine: `sox in.wav out.wav reverb 60 50 100`.
- **The classic drum names are fetched, not bundled.** `s("bd sd hh oh")` rendered with
  the network cut off comes out at -91 dB, silence. Strudel pulls them from github at
  render time. Synths and a few synthesized drums are compiled in and do work offline.
- Renders are otherwise deterministic: the same section rendered twice is byte identical.

## Stack

- **[Strudel](https://strudel.cc)** for the patterns, same language as strudel.cc.
- **[wirbel](https://github.com/jasikpark/wirbel)** renders a `.strudel` file to wav
  through a headless Chromium and an offline audio context.
- **[@strudel/sampler](https://strudel.cc/learn/samples/)** serves `samples/`.
- **ffmpeg** for everything after the render.
- **nix** provides node, chromium, go, sox and rubberband per command.

## The tape room

`bin/shell` drops you in a shell with sox and rubberband alongside ffmpeg and mpv.

```bash
ffmpeg -i a.wav -af "afade=t=in:d=2,afade=t=out:st=30:d=4" out.wav   # fades
ffmpeg -i a.wav -i b.wav -filter_complex amix=inputs=2 out.wav       # layer
ffmpeg -i a.wav -af "volume=-6dB" out.wav                            # level
ffmpeg -i a.wav -af loudnorm=I=-14 out.wav                           # master level
ffmpeg -ss 4 -t 12 -i a.wav out.wav                                  # trim
ffmpeg -f lavfi -i anullsrc=r=44100:cl=stereo -t 3 silence.wav       # silence
ffmpeg -i final.wav -b:a 320k final.mp3                              # publish

sox a.wav out.wav reverb 60 50 100                                   # real reverb
sox a.wav out.wav tremolo 6 40                                       # tremolo
sox a.wav -n spectrogram -o spec.png                                 # picture

rubberband -t 1.25 a.wav out.wav                                     # 25% slower, same pitch
rubberband -p -5 a.wav out.wav                                       # down five semitones
```

Time stretching with `rubberband` beats `atempo`, and pitching with `-p` beats
`asetrate`, on anything tonal.

## Later

- **[dough](https://dough.strudel.cc/)** is a single C file audio engine built after
  superdough, the engine Strudel uses today. It runs as a native binary controlled by
  messages, which would mean rendering with no browser at all. No prebuilt binaries yet,
  so it has to be compiled from the Codeberg source. Worth trying as a second backend
  behind `bin/render`, with everything above unchanged.
