package partitioner

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/rancher-sandbox/elemental-cli/pkg/types/v1"
	"regexp"
	"strconv"
	"strings"
)

type PartedCall struct {
	dev       string
	wipe      bool
	parts     []*Partition
	deletions []int
	label     string
	runner    v1.Runner
}

// We only manage sizes in sectors unit for the Partition structre in parted wrapper
// FileSystem here is only used by parted to determine the partition ID or type
type Partition struct {
	Number     int
	StartS     uint
	SizeS      uint
	PLabel     string
	FileSystem string
}

func NewPartedCall(dev string, runner v1.Runner) *PartedCall {
	return &PartedCall{dev: dev, wipe: false, parts: []*Partition{}, deletions: []int{}, label: "", runner: runner}
}

func (pc PartedCall) optionsBuilder() []string {
	opts := []string{}
	label := pc.label
	match, _ := regexp.MatchString("msdos|gpt", label)
	// Fallback to gpt if label is empty or invalid
	if !match {
		label = "gpt"
	}

	if pc.wipe {
		opts = append(opts, "mklabel", label)
	}

	for _, partnum := range pc.deletions {
		opts = append(opts, "rm", fmt.Sprintf("%d", partnum))
	}

	for _, part := range pc.parts {
		var pLabel string
		if label == "gpt" && part.PLabel != "" {
			pLabel = part.PLabel
		} else if label == "gpt" {
			pLabel = fmt.Sprintf("part%d", part.Number)
		} else {
			pLabel = "primary"
		}

		opts = append(opts, "mkpart", pLabel)

		if part.FileSystem != "" {
			opts = append(opts, part.FileSystem)
		}

		if part.SizeS == 0 {
			// Size set to zero means is interperted as all space available
			opts = append(opts, fmt.Sprintf("%d", part.StartS), "100%")
		} else {
			opts = append(opts, fmt.Sprintf("%d", part.StartS), fmt.Sprintf("%d", part.StartS+part.SizeS-1))
		}
	}

	if len(opts) == 0 {
		return nil
	}

	return append([]string{"--script", "--machine", "--", pc.dev, "unit", "s"}, opts...)
}

func (pc *PartedCall) WriteChanges() (string, error) {
	opts := pc.optionsBuilder()
	if len(opts) == 0 {
		return "", nil
	}

	out, err := pc.runner.Run("parted", opts...)
	pc.wipe = false
	pc.parts = []*Partition{}
	pc.deletions = []int{}
	return string(out), err
}

func (pc *PartedCall) SetPartitionTableLabel(label string) {
	pc.label = label
}

func (pc *PartedCall) CreatePartition(p *Partition) {
	pc.parts = append(pc.parts, p)
}

func (pc *PartedCall) DeletePartition(num int) {
	pc.deletions = append(pc.deletions, num)
}

func (pc *PartedCall) WipeTable(wipe bool) {
	pc.wipe = wipe
}

func (pc PartedCall) Print() (string, error) {
	out, err := pc.runner.Run("parted", "--script", "--machine", "--", pc.dev, "unit", "s", "print")
	return string(out), err
}

// Parses the output of a PartedCall.Print call
func (pc PartedCall) parseHeaderFields(printOut string, field int) (string, error) {
	re := regexp.MustCompile("^(.*):(\\d+)s:(.*):(\\d+):(\\d+):(.*):(.*):(.*);$")

	scanner := bufio.NewScanner(strings.NewReader(strings.TrimSpace(printOut)))
	for scanner.Scan() {
		match := re.FindStringSubmatch(strings.TrimSpace(scanner.Text()))
		if match != nil {
			return match[field], nil
		}
	}
	return "", errors.New("Failed parsing parted header data")
}

// Parses the output of a PartedCall.Print call
func (pc PartedCall) GetLastSector(printOut string) (uint, error) {
	field, err := pc.parseHeaderFields(printOut, 2)
	if err != nil {
		return 0, errors.New("Failed parsing last sector")
	}
	lastSec, err := strconv.ParseUint(field, 10, 0)
	return uint(lastSec), err
}

// Parses the output of a PartedCall.Print call
func (pc PartedCall) GetSectorSize(printOut string) (uint, error) {
	field, err := pc.parseHeaderFields(printOut, 4)
	if err != nil {
		return 0, errors.New("Failed parsing sector size")
	}
	secSize, err := strconv.ParseUint(field, 10, 0)
	return uint(secSize), err
}

// Parses the output of a PartedCall.Print call
func (pc PartedCall) GetPartitionTableLabel(printOut string) (string, error) {
	return pc.parseHeaderFields(printOut, 6)
}

// Parses the output of a GdiskCall.Print call
func (pc PartedCall) GetPartitions(printOut string) []Partition {
	re := regexp.MustCompile("^(\\d+):(\\d+)s:(\\d+)s:(\\d+)s:(.*):(.*):(.*);$")
	var start uint
	var end uint
	var size uint
	var pLabel string
	var partNum int
	var partitions []Partition

	scanner := bufio.NewScanner(strings.NewReader(strings.TrimSpace(printOut)))
	for scanner.Scan() {
		match := re.FindStringSubmatch(strings.TrimSpace(scanner.Text()))
		if match != nil {
			partNum, _ = strconv.Atoi(match[1])
			parsed, _ := strconv.ParseUint(match[2], 10, 0)
			start = uint(parsed)
			parsed, _ = strconv.ParseUint(match[3], 10, 0)
			end = uint(parsed)
			size = end - start + 1
			pLabel = match[6]

			partitions = append(partitions, Partition{
				Number:     partNum,
				StartS:     start,
				SizeS:      size,
				PLabel:     pLabel,
				FileSystem: "",
			})
		}
	}

	return partitions
}
