package main

import (
	"bufio"
	"fmt"
	"github.com/mpetavy/common"
	"slices"
	"strconv"
	"strings"
)

type (
	Hl7Message struct {
		SegmentSeparator      string `json:"segmentSeparator"`
		FieldSeparator        string `json:"fieldSeparator"`
		CompositeSeparator    string `json:"compositeSeparator"`
		RepeatSeparator       string `json:"repeatSeparator"`
		EscapeSeparator       string `json:"escapeSeparator"`
		SubCompositeSeparator string `json:"subCompositeSeparator"`
		Segments              []Hl7Segment
	}
	Hl7Segment struct {
		sep    *string
		Fields []Hl7Field `json:"fields"`
	}
	Hl7Field struct {
		sep     *string
		Repeats []Hl7Repeat `json:"repeats"`
	}
	Hl7Repeat struct {
		sep        *string
		Composites []Hl7Composite `json:"composites"`
	}
	Hl7Composite struct {
		sep           *string
		SubComposites []Hl7SubComposite `json:"subComposites"`
	}
	Hl7SubComposite struct {
		Values []string
	}
)

func (x Hl7Segment) String() string {
	sb := strings.Builder{}
	for i, y := range x.Fields {
		if i > 0 {
			sb.WriteString(*x.sep)
		}

		sb.WriteString(y.String())
	}

	return sb.String()
}

func (x Hl7Field) String() string {
	sb := strings.Builder{}
	for i, y := range x.Repeats {
		if i > 0 {
			sb.WriteString(*x.sep)
		}

		sb.WriteString(y.String())
	}

	return sb.String()
}

func (x Hl7Repeat) String() string {
	sb := strings.Builder{}
	for i, y := range x.Composites {
		if i > 0 {
			sb.WriteString(*x.sep)
		}

		sb.WriteString(y.String())
	}

	return sb.String()
}

func (x Hl7Composite) String() string {
	sb := strings.Builder{}
	for i, y := range x.SubComposites {
		if i > 0 {
			sb.WriteString(*x.sep)
		}

		sb.WriteString(y.String())
	}

	return sb.String()
}

func (x Hl7SubComposite) String() string {
	return strings.Join(x.Values, "")
}

func split(s string, delim string, escape string) []string {
	if s == "" {
		return []string{""}
	}

	list := []string{}

	sb := strings.Builder{}
	for i := 0; i < len(s); i++ {
		if s[i:i+1] == delim && (i == 0 || s[i-1:i] != escape) {
			list = append(list, sb.String())

			sb.Reset()

			continue
		}

		sb.WriteString(s[i : i+1])
	}

	if sb.Len() > 0 {
		list = append(list, sb.String())
	}

	return list
}

func NewHL7Message(ba []byte) (*Hl7Message, error) {
	if len(ba) == 0 {
		return nil, fmt.Errorf("no data")
	}

	hl7Msg := &Hl7Message{}

	msg := strings.TrimSpace(string(ba))
	msg = strings.ReplaceAll(msg, "\r\n", "\r")
	msg = strings.ReplaceAll(msg, "\n", "\r")

	if !strings.HasSuffix(msg, "\r") {
		msg = msg + "\r"
	}

	if !strings.HasPrefix(msg, "MSH") {
		return nil, fmt.Errorf("no HL7 message")
	}

	hl7Msg.SegmentSeparator = "\r"
	hl7Msg.FieldSeparator = msg[3:4]
	hl7Msg.CompositeSeparator = msg[4:5]
	hl7Msg.RepeatSeparator = msg[5:6]
	hl7Msg.EscapeSeparator = msg[6:7]
	hl7Msg.SubCompositeSeparator = msg[7:8]

	withCR, err := common.NewSeparatorSplitFunc(nil, []byte("\r"), true)
	if common.Error(err) {
		return nil, err
	}

	scanner := bufio.NewScanner(strings.NewReader(msg))
	scanner.Split(withCR)
	for scanner.Scan() {
		line := scanner.Text()

		segment := Hl7Segment{sep: &hl7Msg.FieldSeparator}
		fieldItems := split(line, hl7Msg.FieldSeparator, hl7Msg.EscapeSeparator)
		for _, fieldItem := range fieldItems {
			field := Hl7Field{sep: &hl7Msg.RepeatSeparator}

			repeatItems := split(fieldItem, hl7Msg.RepeatSeparator, hl7Msg.EscapeSeparator)
			for _, repeatItem := range repeatItems {
				repeat := Hl7Repeat{sep: &hl7Msg.CompositeSeparator}

				compositeItems := split(repeatItem, hl7Msg.CompositeSeparator, hl7Msg.EscapeSeparator)
				for _, compositeItem := range compositeItems {
					composite := Hl7Composite{sep: &hl7Msg.SubCompositeSeparator}

					subCompositeItems := split(compositeItem, hl7Msg.SubCompositeSeparator, hl7Msg.EscapeSeparator)
					for _, subCompositeItem := range subCompositeItems {
						subComposite := Hl7SubComposite{}

						subComposite.Values = append(subComposite.Values, subCompositeItem)

						composite.SubComposites = append(composite.SubComposites, subComposite)
					}

					repeat.Composites = append(repeat.Composites, composite)
				}

				field.Repeats = append(field.Repeats, repeat)
			}

			segment.Fields = append(segment.Fields, field)
		}

		hl7Msg.Segments = append(hl7Msg.Segments, segment)
	}

	return hl7Msg, nil
}

func (hl7Msg Hl7Message) GetValue(location string) (string, error) {
	var result string

	err := common.Catch(func() error {
		ids := common.Split(strings.ReplaceAll(location, "-", "."), ".")

		indices := make([]int, 4)
		for i, id := range ids {
			if i == 0 {
				continue
			}

			indices[i], _ = strconv.Atoi(id)
			if i > 1 {
				indices[i]--
			}
		}

		p := slices.IndexFunc(hl7Msg.Segments, func(segment Hl7Segment) bool {
			s := segment.Fields[0].Repeats[0].Composites[0].SubComposites[0].String()

			return s == ids[0]
		})

		indices[0] = p

		switch len(ids) {
		default:
			result = hl7Msg.Segments[p].String()
		case 2:
			result = hl7Msg.Segments[p].Fields[indices[1]].String()
		case 3:
			result = hl7Msg.Segments[p].Fields[indices[1]].Repeats[0].Composites[indices[2]].String()
		case 4:
			result = hl7Msg.Segments[p].Fields[indices[1]].Repeats[0].Composites[indices[2]].SubComposites[indices[3]].String()
		}

		return nil
	})

	if common.DebugError(err) {
		return "", fmt.Errorf("invalid location: %s", location)
	}

	return result, nil
}
