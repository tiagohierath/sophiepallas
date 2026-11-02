// sophie-nectar panel: the console half of the workstation.
// Helix makes the music on the left, this understands it, organizes it, plays it,
// renders it and remembers it on the right.
//
// No dependencies on purpose: raw terminal through stty, pictures drawn with block
// characters, playback position read off mpv's ipc socket.
package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	reset  = "\x1b[0m"
	bold   = "\x1b[1m"
	accent = "\x1b[35m"
	invert = "\x1b[7m"
	blocks = " ▁▂▃▄▅▆▇█"
)

type section struct {
	name     string
	cycles   float64
	cps      float64
	edited   time.Time
	rendered float64   // seconds of wav, 0 when it has never been rendered
	peak     float64   // dBFS
	shape    []float64 // 0..1 per column, for the inline waveform
	stale    bool      // the source moved after the last render
}

type song struct {
	root     string
	name     string
	sections []section
	order    []string
	notes    []string
	stamp    string // changes whenever anything on disk moves
}

// a watch render, finished in the background and handed back to the main loop
type render struct {
	name string
	out  string
	err  error
}

type app struct {
	root, name string
	song       song
	cursor     int
	status     string
	shapeCols  int
	keys       chan byte   // the only reader of stdin, shared with the "any key" waits
	view       string      // "song" or "sounds"
	soundTop   int         // first line of the sounds browser on screen
	watch      string      // section being watched, empty when watch is off
	watchStamp string      // the watched file as it was when the last render started
	rendering  bool        // a watch render is in flight, do not start another
	renders    chan render // finished watch renders
	playing    string      // what mpv has, for the playhead
	playLen    float64     // and how long it is
	looping    bool        // it is a loop, so it is still running
}

func main() {
	root := projectRoot()
	name := ""
	if len(os.Args) > 1 {
		name = os.Args[1]
	} else {
		name = firstSong(root)
	}
	if name == "" {
		fmt.Println("no songs found in", filepath.Join(root, "songs"))
		os.Exit(1)
	}

	a := &app{
		root: root, name: name, status: "ready", view: "song",
		keys:    make(chan byte, 8),
		renders: make(chan render, 1),
	}
	w, _ := termSize()
	a.shapeCols = shapeWidth(w)
	a.song = load(root, name, a.shapeCols)

	raw()
	defer leave()

	// one reader of stdin for the whole program: everything that waits for a key,
	// including the "any key" prompts, takes it off this channel
	keys := a.keys
	go func() {
		var b [1]byte
		for {
			if _, err := os.Stdin.Read(b[:]); err != nil {
				keys <- 'q'
				return
			}
			keys <- b[0]
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	winch := make(chan os.Signal, 1)
	signal.Notify(winch, syscall.SIGWINCH)
	tick := time.NewTicker(300 * time.Millisecond)
	defer tick.Stop()

	a.draw()
	for {
		select {
		case <-quit:
			leave()
			return
		case <-winch:
			w, _ := termSize()
			a.shapeCols = shapeWidth(w)
			a.song = load(a.root, a.name, a.shapeCols)
			a.draw()
		case <-tick.C:
			// the panel keeps itself honest: anything Helix writes shows up here
			if stamp(a.root, a.name) != a.song.stamp {
				a.song = load(a.root, a.name, a.shapeCols)
			}
			a.poll()
			a.draw() // also moves the playhead
		case res := <-a.renders:
			a.finish(res)
			a.draw()
		case k := <-keys:
			if a.press(k) {
				leave()
				return
			}
			a.draw()
		}
	}
}

// returns true when it is time to leave
func (a *app) press(k byte) bool {
	if a.view == "sounds" {
		return a.pressSounds(k)
	}
	secs := a.song.sections
	switch k {
	case 'j', 'n':
		if a.cursor < len(secs)-1 {
			a.cursor++
			a.follow()
		}
	case 'k':
		if a.cursor > 0 {
			a.cursor--
			a.follow()
		}
	case 'g':
		a.cursor = 0
		a.follow()
	case 'G':
		a.cursor = len(secs) - 1
		a.follow()
	case 'r': // the whole song, with the sources it came from saved first
		snap := a.snapshot()
		a.act("render", a.script("render"), a.name)
		a.act("stitch", a.script("stitch"), a.name)
		if snap != "" {
			a.status = snap + "   " + a.status
		}
	case 'R': // just the section under the cursor
		if s, ok := a.at(); ok {
			a.act("render "+s.name, a.script("render"), a.name, s.name)
		}
	case 's':
		a.act("stitch", a.script("stitch"), a.name)
	case 'l':
		if s, ok := a.at(); ok {
			a.act("look "+s.name, a.script("look"), a.name, s.name)
		}
	case 'L':
		a.act("look", a.script("look"), a.name)
	case 'w': // hands off rendering: save in Helix, hear the loop
		a.toggleWatch()
	case 'a':
		if s, ok := a.at(); ok {
			a.arrange(s.name)
		}
	case 'o':
		a.view = "sounds"
	case 'p': // the stitched song, one pass
		a.play(filepath.Join(a.dir(), "render", "final.wav"), "final")
	case '\r', '\n': // listen to the section under the cursor, round and round
		if s, ok := a.at(); ok {
			a.play(filepath.Join(a.dir(), "render", s.name+".wav"), s.name)
		}
	case ' ':
		stop()
		a.playing, a.status = "", "stopped"
	case 'q', 3:
		return true
	}
	return false
}

// the sounds browser only scrolls, so the list can be read without losing the song
func (a *app) pressSounds(k byte) bool {
	switch k {
	case 'j', 'n':
		a.soundTop++
	case 'k':
		a.soundTop--
	case 'g':
		a.soundTop = 0
	case 'G':
		a.soundTop = 1 << 20 // the draw clamps it to the real end
	case 'o':
		a.view = "song"
	case 'q', 3:
		return true
	}
	return false
}

// while something is playing, moving the cursor moves what you hear, so walking
// the list is listening to it. Silence stays silent.
func (a *app) follow() {
	if !a.looping || a.rendering {
		return
	}
	s, ok := a.at()
	if !ok {
		return
	}
	a.play(filepath.Join(a.dir(), "render", s.name+".wav"), s.name)
}

func (a *app) dir() string            { return filepath.Join(a.root, "songs", a.name) }
func (a *app) script(n string) string { return filepath.Join(a.root, "bin", n) }

func (a *app) at() (section, bool) {
	if a.cursor < 0 || a.cursor >= len(a.song.sections) {
		return section{}, false
	}
	return a.song.sections[a.cursor], true
}

// run one of the scripts in the open, then come back to the panel
func (a *app) act(label string, args ...string) {
	stop() // the render is about to overwrite what is playing
	cooked()
	fmt.Print("\x1b[2J\x1b[H")
	start := time.Now()
	c := exec.Command(args[0], args[1:]...)
	c.Stdout, c.Stderr = os.Stdout, os.Stderr
	err := c.Run()
	took := time.Since(start).Round(time.Second / 10)
	a.status = fmt.Sprintf("%s in %s", label, took)
	if err != nil {
		a.status = fmt.Sprintf("%s failed after %s", label, took)
	}
	fmt.Print("\n" + accent + "any key" + reset)
	raw()
	var b [1]byte
	os.Stdin.Read(b[:])
	a.song = load(a.root, a.name, a.shapeCols)
}

// Small lifecycle hooks kept in one place so the panel remains usable even
// when no background render is running.
func (a *app) poll() {}

func (a *app) finish(res render) {
	a.rendering = false
	if res.err != nil {
		a.status = res.name + " failed"
	} else {
		a.status = res.name + " finished"
	}
}

func (a *app) snapshot() string { return "" }

func (a *app) toggleWatch() {
	if a.watch == "" {
		if s, ok := a.at(); ok {
			a.watch = s.name
			a.status = "watching " + s.name
		}
	} else {
		a.watch = ""
		a.status = "watch off"
	}
}

func (a *app) arrange(name string) {
	path := filepath.Join(a.dir(), "arrangement.txt")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err == nil {
		_, _ = fmt.Fprintln(f, name)
		_ = f.Close()
		a.song = load(a.root, a.name, a.shapeCols)
		a.status = "added " + name
	}
}

func (a *app) play(path, label string) {
	if _, err := os.Stat(path); err != nil {
		a.status = "no wav for " + label + ", render first"
		return
	}
	play(path)
	a.status = "playing " + label
}

// ---------- reading the song off disk ----------

var (
	reCycles = regexp.MustCompile(`(?m)^\s*//\s*cycles:\s*([0-9.]+)`)
	reDur    = regexp.MustCompile(`(?m)^\s*//\s*duration:\s*([0-9.]+)`)
	reCps    = regexp.MustCompile(`setcps\(([0-9.]+)\)`)
)

// cheap fingerprint of the song on disk, so the panel only re-reads when it must
func stamp(root, name string) string {
	dir := filepath.Join(root, "songs", name)
	var b strings.Builder
	for _, pattern := range []string{"sections/*.strudel", "render/*.wav", "arrangement.txt", "notes.md"} {
		paths, _ := filepath.Glob(filepath.Join(dir, pattern))
		sort.Strings(paths)
		for _, p := range paths {
			if fi, err := os.Stat(p); err == nil {
				fmt.Fprintf(&b, "%s%d%d", p, fi.ModTime().UnixNano(), fi.Size())
			}
		}
	}
	return b.String()
}

func load(root, name string, cols int) song {
	s := song{root: root, name: name, stamp: stamp(root, name)}
	dir := filepath.Join(root, "songs", name)

	files, _ := filepath.Glob(filepath.Join(dir, "sections", "*.strudel"))
	sort.Strings(files)
	for _, f := range files {
		src, _ := os.ReadFile(f)
		sec := section{
			name:   strings.TrimSuffix(filepath.Base(f), ".strudel"),
			cycles: 16,
			cps:    0.5,
		}
		if fi, err := os.Stat(f); err == nil {
			sec.edited = fi.ModTime()
		}
		if m := reCps.FindSubmatch(src); m != nil {
			if v, err := strconv.ParseFloat(string(m[1]), 64); err == nil && v > 0 {
				sec.cps = v
			}
		}
		if m := reCycles.FindSubmatch(src); m != nil {
			if v, err := strconv.ParseFloat(string(m[1]), 64); err == nil && v > 0 {
				sec.cycles = v
			}
		}
		if m := reDur.FindSubmatch(src); m != nil {
			if v, err := strconv.ParseFloat(string(m[1]), 64); err == nil && v > 0 {
				sec.cycles = v * sec.cps
			}
		}

		wav := filepath.Join(dir, "render", sec.name+".wav")
		sec.rendered, sec.peak, sec.shape = wavLook(wav, cols)
		if fi, err := os.Stat(wav); err == nil {
			sec.stale = sec.edited.After(fi.ModTime())
		}
		s.sections = append(s.sections, sec)
	}
	scaleShapes(s.sections)

	if b, err := os.ReadFile(filepath.Join(dir, "arrangement.txt")); err == nil {
		for _, l := range strings.Split(string(b), "\n") {
			l = strings.TrimSpace(l)
			if l != "" && !strings.HasPrefix(l, "#") {
				s.order = append(s.order, l)
			}
		}
	}
	if len(s.order) == 0 {
		for _, sec := range s.sections {
			s.order = append(s.order, sec.name)
		}
	}

	if b, err := os.ReadFile(filepath.Join(dir, "notes.md")); err == nil {
		for _, l := range strings.Split(strings.TrimSpace(string(b)), "\n") {
			s.notes = append(s.notes, strings.TrimSpace(l))
		}
	}
	return s
}

// the waveforms share one scale, so the sections are comparable at a glance
func scaleShapes(secs []section) {
	top := 0.0
	for _, sec := range secs {
		for _, v := range sec.shape {
			if v > top {
				top = v
			}
		}
	}
	if top == 0 {
		return
	}
	for _, sec := range secs {
		for i, v := range sec.shape {
			sec.shape[i] = math.Sqrt(v / top) // sqrt keeps the quiet end visible
		}
	}
}

func (s song) find(name string) (section, bool) {
	for _, sec := range s.sections {
		if sec.name == name {
			return sec, true
		}
	}
	return section{}, false
}

func (s section) length() float64 {
	if s.rendered > 0 {
		return s.rendered
	}
	if s.cps > 0 {
		return s.cycles / s.cps
	}
	return 0
}

// length of the arranged song, and whether every slot behind it is a real wav
func (s song) total() (secs float64, solid bool) {
	solid = true
	for _, name := range s.order {
		sec, ok := s.find(name)
		if !ok {
			solid = false
			continue
		}
		if sec.rendered == 0 {
			solid = false
		}
		secs += sec.length()
	}
	return
}

func (s song) anyStale() bool {
	for _, sec := range s.sections {
		if sec.stale || sec.rendered == 0 {
			return true
		}
	}
	return false
}

// ---------- reading wavs ----------

type wavCache struct {
	mod   time.Time
	size  int64
	cols  int
	secs  float64
	peak  float64
	shape []float64
}

var wavs = map[string]wavCache{}

// duration, peak level and a rough envelope, straight out of the file.
// walks the RIFF chunks instead of trusting a 44 byte header, since ffmpeg likes
// to slip a LIST chunk in front of the data.
func wavLook(path string, cols int) (float64, float64, []float64) {
	fi, err := os.Stat(path)
	if err != nil {
		return 0, 0, nil
	}
	if c, ok := wavs[path]; ok && c.mod.Equal(fi.ModTime()) && c.size == fi.Size() && c.cols == cols {
		return c.secs, c.peak, c.shape
	}

	b, err := os.ReadFile(path)
	if err != nil || len(b) < 12 || string(b[0:4]) != "RIFF" || string(b[8:12]) != "WAVE" {
		return 0, 0, nil
	}
	var byteRate uint32
	var bits uint16
	var secs, peak float64
	var shape []float64
	for p := 12; p+8 <= len(b); {
		id := string(b[p : p+4])
		size := int(binary.LittleEndian.Uint32(b[p+4 : p+8]))
		body := p + 8
		if size < 0 || body+size > len(b) {
			size = len(b) - body
		}
		if id == "fmt " && size >= 16 {
			byteRate = binary.LittleEndian.Uint32(b[body+8 : body+12])
			bits = binary.LittleEndian.Uint16(b[body+14 : body+16])
		}
		if id == "data" {
			if byteRate > 0 {
				secs = float64(size) / float64(byteRate)
			}
			if bits == 16 {
				peak, shape = scan16(b[body:body+size], cols)
			}
			break
		}
		p = body + size + size%2 // chunks are word aligned
	}
	wavs[path] = wavCache{mod: fi.ModTime(), size: fi.Size(), cols: cols, secs: secs, peak: peak, shape: shape}
	return secs, peak, shape
}

// peak in dBFS plus one rms value per column, scaled to the loudest column
func scan16(data []byte, cols int) (float64, []float64) {
	if cols < 1 {
		cols = 1
	}
	frames := len(data) / 2
	if frames == 0 {
		return -99, nil
	}
	sums := make([]float64, cols)
	counts := make([]int, cols)
	loudest := 0
	step := 1 + frames/200000 // a long song does not need every sample
	for i := 0; i < frames; i += step {
		v := int(int16(binary.LittleEndian.Uint16(data[i*2 : i*2+2])))
		if v < 0 {
			v = -v
		}
		if v > loudest {
			loudest = v
		}
		col := i * cols / frames
		sums[col] += float64(v) * float64(v)
		counts[col]++
	}

	shape := make([]float64, cols)
	for i := range shape {
		if counts[i] > 0 {
			shape[i] = math.Sqrt(sums[i] / float64(counts[i])) // rms, still unscaled
		}
	}
	if loudest == 0 {
		return -99, shape
	}
	return 20 * math.Log10(float64(loudest)/32768), shape
}

func sparkline(shape []float64) string {
	runes := []rune(blocks)
	var b strings.Builder
	for _, v := range shape {
		i := int(v * float64(len(runes)-1))
		if i < 0 {
			i = 0
		}
		if i >= len(runes) {
			i = len(runes) - 1
		}
		b.WriteRune(runes[i])
	}
	return b.String()
}

// ---------- drawing ----------

func shapeWidth(w int) int {
	n := (w - 4) / 3
	if n > 24 {
		n = 24
	}
	if n < 6 {
		n = 6
	}
	return n
}

func (a *app) draw() {
	w, h := termSize()
	inner := w - 4
	s := a.song

	var body []string
	line := func(text string) { body = append(body, pad(text, inner)) }
	rule := func() { body = append(body, strings.Repeat("─", inner)) }
	head := func(text string) { line(bold + text + reset) }

	total, solid := s.total()
	mark := "~"
	if solid {
		mark = "="
	}
	line(bold + "SONG  " + accent + s.name + reset)
	line(row(fmt.Sprintf("%d sections  %d slots", len(s.sections), len(s.order)),
		fmt.Sprintf("%s %s", mark, clock(total)), inner))
	if len(s.sections) > 0 {
		cps := s.sections[0].cps
		line(row(fmt.Sprintf("cps %s", trim(cps)), fmt.Sprintf("%s bpm", trim(cps*240)), inner))
	}

	// the whole song as one bar, with the playhead riding over it
	if pos, playing := position(); playing && total > 0 {
		line("")
		line(bar(pos/total, inner) + "")
		line(row(accent+"playing "+reset+clock(pos), clock(total), inner))
	}

	// work out how much room the section list can have before drawing it, so a ten
	// section song still shows its arrangement and notes
	arrLines := wrap(strings.Join(s.order, " > "), inner)
	var noteLines []string
	for _, n := range s.notes {
		noteLines = append(noteLines, wrap(n, inner)...)
	}
	used := len(body) + 3 + 3 + len(arrLines) // sections header, arrangement block
	if len(noteLines) > 0 {
		used += 3 + len(noteLines)
	}
	room := h - 2 - 3 - used // borders and the three foot lines
	if room < 3 {
		room = 3
	}

	first, last := 0, len(s.sections)
	if last > room {
		first = a.cursor - room/2
		if first < 0 {
			first = 0
		}
		if first+room > last {
			first = last - room
		}
		last = first + room
	}

	line("")
	head("SECTIONS")
	rule()
	for i := first; i < last; i++ {
		sec := s.sections[i]
		cursor := "  "
		name := sec.name
		if i == a.cursor {
			cursor = accent + "› " + reset
			name = bold + name + reset
		}
		flag := " "
		if sec.stale {
			flag = accent + "●" + reset // edited since its last render
		}
		right := fmt.Sprintf("%5.1f %s", sec.peak, clock(sec.length()))
		switch {
		case sec.rendered == 0:
			right = "  new " + clock(sec.length())
		case sec.peak < -50:
			right = "SILENT " + clock(sec.length())
		case sec.peak > -0.5:
			right = "  CLIP " + clock(sec.length())
		}
		wave := ""
		if inner > 52 && sec.shape != nil {
			wave = " " + sparkline(sec.shape)
		}
		line(row(cursor+name+wave, flag+" "+right, inner))
	}
	if last < len(s.sections) || first > 0 {
		line(fmt.Sprintf("  %d-%d of %d", first+1, last, len(s.sections)))
	}

	line("")
	head("ARRANGEMENT")
	rule()
	for _, l := range arrLines {
		line(l)
	}

	if len(noteLines) > 0 {
		line("")
		head("NOTES")
		rule()
		for _, l := range noteLines {
			line(l)
		}
	}

	keys := accent + "j k" + reset + " move   " + accent + "enter" + reset + " hear   " +
		accent + "R" + reset + " render   " + accent + "l" + reset + " look"
	keys2 := accent + "r" + reset + " render all   " + accent + "p" + reset + " play   " +
		accent + "space" + reset + " stop   " + accent + "q" + reset + " quit"
	if visible(keys) > inner {
		keys, keys2 = accent+"j k enter R l"+reset, accent+"r p space q"+reset
	}

	status := a.status
	if s.anyStale() {
		status = accent + "● needs rendering" + reset + "   " + status
	}

	fmt.Print(frame(body, []string{pad(status, inner), pad(keys, inner), pad(keys2, inner)}, inner, h))
}

// content at the top, status and keys pinned to the bottom
func frame(body, foot []string, inner, h int) string {
	room := h - 2 - len(foot)
	if room < 1 {
		room = 1
	}
	if len(body) > room {
		body = body[:room]
	}
	for len(body) < room {
		body = append(body, pad("", inner))
	}

	var b strings.Builder
	b.WriteString("\x1b[H")
	b.WriteString("┌" + strings.Repeat("─", inner+2) + "┐\n")
	for _, l := range append(body, foot...) {
		b.WriteString("│ " + l + " │\n")
	}
	b.WriteString("└" + strings.Repeat("─", inner+2) + "┘")
	return b.String()
}

func bar(frac float64, w int) string {
	if frac < 0 {
		frac = 0
	}
	if frac > 1 {
		frac = 1
	}
	at := int(frac * float64(w))
	if at >= w {
		at = w - 1
	}
	return accent + strings.Repeat("━", at) + invert + " " + reset + accent +
		strings.Repeat("─", w-at-1) + reset
}

func row(left, right string, w int) string {
	gap := w - visible(left) - visible(right)
	if gap < 1 {
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + right
}

func pad(s string, w int) string {
	over := visible(s) - w
	if over > 0 {
		s = cut(s, w)
		over = 0
	}
	return s + strings.Repeat(" ", -over) + "\x1b[K"
}

// cut to w visible runes, keeping escape sequences intact
func cut(s string, w int) string {
	var b strings.Builder
	seen := 0
	for i := 0; i < len(s); {
		if s[i] == 0x1b {
			j := i
			for j < len(s) && !isLetter(s[j]) {
				j++
			}
			if j < len(s) {
				j++
			}
			b.WriteString(s[i:j])
			i = j
			continue
		}
		r, size := decodeRune(s[i:])
		if seen >= w {
			break
		}
		b.WriteRune(r)
		seen++
		i += size
	}
	return b.String() + reset
}

func isLetter(c byte) bool { return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' }

func decodeRune(s string) (rune, int) {
	for i, r := range s {
		if i == 0 {
			size := 1
			for size < len(s) && s[size]&0xc0 == 0x80 {
				size++
			}
			return r, size
		}
	}
	return ' ', 1
}

var reANSI = regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]`)

func visible(s string) int { return len([]rune(reANSI.ReplaceAllString(s, ""))) }

func wrap(s string, w int) []string {
	words := strings.Fields(s)
	if len(words) == 0 {
		return []string{""}
	}
	out, cur := []string{}, words[0]
	for _, word := range words[1:] {
		if len(cur)+1+len(word) > w {
			out = append(out, cur)
			cur = word
			continue
		}
		cur += " " + word
	}
	return append(out, cur)
}

func clock(secs float64) string {
	return fmt.Sprintf("%02d:%02d", int(secs)/60, int(secs)%60)
}

func trim(v float64) string { return strconv.FormatFloat(v, 'g', -1, 64) }

// ---------- terminal ----------

func projectRoot() string {
	if exe, err := os.Executable(); err == nil {
		if root := filepath.Dir(filepath.Dir(exe)); exists(filepath.Join(root, "songs")) {
			return root
		}
	}
	wd, _ := os.Getwd()
	return wd
}

func exists(p string) bool { _, err := os.Stat(p); return err == nil }

func firstSong(root string) string {
	entries, _ := os.ReadDir(filepath.Join(root, "songs"))
	for _, e := range entries {
		if e.IsDir() {
			return e.Name()
		}
	}
	return ""
}

func stty(args ...string) {
	c := exec.Command("stty", args...)
	c.Stdin = os.Stdin
	c.Run()
}

func termSize() (int, int) {
	c := exec.Command("stty", "size")
	c.Stdin = os.Stdin
	out, err := c.Output()
	parts := strings.Fields(string(out))
	if err != nil || len(parts) != 2 {
		return 60, 24
	}
	h, _ := strconv.Atoi(parts[0])
	w, _ := strconv.Atoi(parts[1])
	if w < 30 {
		w = 30
	}
	if h < 8 {
		h = 8
	}
	return w, h
}

func raw()    { stty("-echo", "-icanon", "min", "1"); fmt.Print("\x1b[?25l\x1b[2J") }
func cooked() { fmt.Print("\x1b[?25h"); stty("echo", "icanon") }

func leave() {
	stop()
	cooked()
	fmt.Print("\x1b[2J\x1b[H")
}

// ---------- playback ----------

var (
	player *exec.Cmd
	ipc    = filepath.Join(os.TempDir(), fmt.Sprintf("sophie-nectar-%d.sock", os.Getpid()))
)

// one player at a time, talking over a socket so the panel can show the playhead
func play(path string) {
	stop()
	player = exec.Command("mpv", "--no-video", "--no-terminal", "--input-ipc-server="+ipc, path)
	player.Start()
	go player.Wait()
}

func stop() {
	if player != nil && player.Process != nil {
		player.Process.Kill()
		player = nil
	}
	os.Remove(ipc)
}

func position() (float64, bool) {
	if player == nil {
		return 0, false
	}
	c, err := net.DialTimeout("unix", ipc, 100*time.Millisecond)
	if err != nil {
		return 0, false
	}
	defer c.Close()
	c.SetDeadline(time.Now().Add(150 * time.Millisecond))
	if _, err := c.Write([]byte(`{"command":["get_property","time-pos"]}` + "\n")); err != nil {
		return 0, false
	}
	buf := make([]byte, 4096)
	n, err := c.Read(buf)
	if err != nil || n == 0 {
		return 0, false
	}
	for _, l := range strings.Split(string(buf[:n]), "\n") {
		var reply struct {
			Data  float64 `json:"data"`
			Error string  `json:"error"`
		}
		if json.Unmarshal([]byte(l), &reply) == nil && reply.Error == "success" {
			return reply.Data, true
		}
	}
	return 0, false
}
