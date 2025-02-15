// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"testing"
	"testing/quick"

	"github.com/aws/lightsailctl/internal"
	"github.com/aws/lightsailctl/internal/cs"
)

func TestInputVersion(t *testing.T) {
	var tests = []struct {
		pass  bool
		input string
	}{
		{input: `{"inputVersion": ""}`},
		{input: `{"inputVersion": "v1"}`},
		{input: `{"inputVersion": "bogus"}`},
		{
			pass:  true,
			input: `{"inputVersion": "55"}`,
		},
	}

	for _, test := range tests {
		_, err := parseInput(strings.NewReader(test.input))
		if test.pass {
			if err != nil {
				t.Errorf("%s: %v", test.input, err)
			}
			continue
		}
		if err == nil {
			t.Errorf("%s: unexpected succeeded", test.input)
		}
	}

	f := func(i int) bool {
		input := fmt.Sprintf(`{"inputVersion": "%v"}`, i)
		parsed, err := parseInput(strings.NewReader(input))
		if i < 0 {
			return err != nil
		}
		if err != nil {
			return false
		}
		return parsed.InputVersion == strconv.Itoa(i)
	}
	if err := quick.Check(f, nil); err != nil {
		t.Error(err)
	}
}

func TestParseInput(t *testing.T) {
	input := `{
		"inputVersion":  "1",
		"operation":     "Whatever",
		"payload":       42,
		"configuration": {"region": "us-west-2", "cliVersion": "2.0.47"}
	}`

	got, err := parseInput(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}

	want := &Input{
		InputVersion:  "1",
		Operation:     "Whatever",
		Payload:       []byte{'4', '2'},
		Configuration: OperationConfig{Region: "us-west-2", CLIVersion: "2.0.47"},
	}

	internal.Assert(t, "parsed input", want, got)
}

func TestParsePushContainerImagePayload(t *testing.T) {
	inputf := `{
		"inputVersion":  "1",
		"operation":     "PushContainerImage",
		"payload":       %s,
		"configuration": {"region": "us-west-2", "cliVersion": "2.0.47"}
	}`

	for i, test := range []struct {
		payload string
		wantErr string
		want    *cs.PushImageInput
	}{
		{
			payload: `{"service": "dyservicev3", "image": "hello:latest"}`,
			wantErr: "push container image: container label is not specified",
		},
		{
			payload: `{"service": "dyservicev3", "label": "david16"}`,
			wantErr: "push container image: container image is not specified",
		},
		{
			payload: `{"image": "hello:latest", "label": "david16"}`,
			wantErr: "push container image: service name is not specified",
		},
		{
			payload: `{"service": "dyservicev3", "image": "hello:latest", "label": "david16"}`,
			want:    &cs.PushImageInput{Service: "dyservicev3", Image: "hello:latest", Label: "david16"},
		},
	} {
		t.Run(strconv.Itoa(i+1), func(t *testing.T) {
			in, err := parseInput(strings.NewReader(fmt.Sprintf(inputf, test.payload)))
			if err != nil {
				t.Error(err)
				return
			}

			got, err := parsePushContainerImagePayload(in.Payload)
			internal.AssertErr(t, test.wantErr, err)
			internal.Assert(t, "PushImageInput", test.want, got)
		})
	}
}

func TestOutputFormat(t *testing.T) {
	for _, test := range []struct {
		output      OutputFormat
		wantErr     string
		wantEncoded string
		ioFailure   bool
	}{
		{
			output:      "json",
			wantEncoded: `{"digest":"sha256:10b8cc432d56da8b61b070f4c7d2543a9ed17c2b23010b43af434fd40e2ca4aa","image":"nginx","pushedImage":":nginx.www.35"}`,
		},
		{
			output: "text",
			wantEncoded: `Digest: sha256:10b8cc432d56da8b61b070f4c7d2543a9ed17c2b23010b43af434fd40e2ca4aa
Image "nginx" registered.
Refer to this image as ":nginx.www.35" in deployments.`,
		},
		{
			output:    "other",
			ioFailure: true,
			wantErr:   `format *cs.PushImageOutput as "text": io: read/write on closed pipe`,
		},
	} {
		t.Run(string(test.output), func(t *testing.T) {
			sb := &strings.Builder{}
			var w io.Writer = sb
			if test.ioFailure {
				// Provoke an IO failure!
				pr, pw := io.Pipe()
				pr.Close()
				w = pw
			}
			err := test.output.encode(w, &cs.PushImageOutput{
				Digest:      "sha256:10b8cc432d56da8b61b070f4c7d2543a9ed17c2b23010b43af434fd40e2ca4aa",
				Image:       "nginx",
				PushedImage: ":nginx.www.35",
			})
			internal.AssertErr(t, test.wantErr, err)
			internal.Assert(t, "encoded output", test.wantEncoded, strings.TrimSpace(sb.String()))
		})
	}
}
