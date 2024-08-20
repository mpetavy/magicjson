package main

import (
	"encoding/json"
	"fmt"
	"github.com/mpetavy/common"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

func TestHL7(t *testing.T) {
	ba, err := os.ReadFile("testdata/sample-hl7.hl7")
	require.NoError(t, err)

	ba, err = common.ToUTF8(ba, "")
	require.NoError(t, err)

	hl7Msg, err := NewHL7Message(ba)
	require.NoError(t, err)

	fmt.Printf("%s\n", hl7Msg)

	j, err := json.MarshalIndent(hl7Msg, "", "    ")
	require.NoError(t, err)

	fmt.Println(string(j))

	type args struct {
		location string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name:    "0",
			args:    args{"PID.0"},
			want:    "PID",
			wantErr: false,
		},
		{
			name:    "1",
			args:    args{"PID.2"},
			want:    "0493575^^^2^ID 1",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := hl7Msg.GetValue(tt.args.location)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetValue() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHL7_1(t *testing.T) {
	ba, err := os.ReadFile("testdata/test.hl7")
	require.NoError(t, err)

	ba, err = common.ToUTF8(ba, "")
	require.NoError(t, err)

	hl7Msg, err := NewHL7Message(ba)
	require.NoError(t, err)

	type args struct {
		location string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name:    "0",
			args:    args{"PID.1"},
			want:    "1",
			wantErr: false,
		},
		{
			name:    "1",
			args:    args{"PID.2"},
			want:    "",
			wantErr: false,
		},
		{
			name:    "2",
			args:    args{"PID.3"},
			want:    "123456^^^DH&MR",
			wantErr: false,
		},
		{
			name:    "3",
			args:    args{"PID.3.4.1"},
			want:    "DH",
			wantErr: false,
		},
		{
			name:    "3",
			args:    args{"PID.3.4.2"},
			want:    "MR",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := hl7Msg.GetValue(tt.args.location)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetValue() got = %v, want %v", got, tt.want)
			}
		})
	}
}
