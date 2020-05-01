package rproxy

import (
	"context"
	"golang.org/x/crypto/acme/autocert"
	"reflect"
	"testing"
)

func Test_memCache_Delete(t *testing.T) {
	type fields struct {
		data map[string][]byte
	}
	type args struct {
		key string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:    "not exists",
			fields:  fields{},
			args:    args{key: "test"},
			wantErr: false,
		},
		{
			name:    "exists",
			fields:  fields{data: map[string][]byte{"test": nil}},
			args:    args{key: "test"},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewMemCache().(*memCache)
			for key, val := range tt.fields.data {
				m.data[key] = val
			}
			if err := m.Delete(context.TODO(), tt.args.key); (err != nil) != tt.wantErr {
				t.Errorf("Delete() error = %v, wantErr %v", err, tt.wantErr)
			}

			if _, ok := m.data[tt.args.key]; ok {
				t.Fatal()
			}
		})
	}
}

func Test_memCache_Get(t *testing.T) {
	type fields struct {
		data map[string][]byte
	}
	type args struct {
		key string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []byte
		wantErr error
	}{
		{
			"not exists",
			fields{},
			args{key: "test"},
			nil,
			autocert.ErrCacheMiss,
		},
		{
			"exists",
			fields{data: map[string][]byte{"test": []byte("hello, world")}},
			args{key: "test"},
			[]byte("hello, world"),
			nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewMemCache().(*memCache)
			for key, val := range tt.fields.data {
				m.data[key] = val
			}
			got, err := m.Get(context.TODO(), tt.args.key)
			if err != tt.wantErr {
				t.Errorf("Get() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Get() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_memCache_Put(t *testing.T) {
	type fields struct {
		data map[string][]byte
	}
	type args struct {
		key  string
		data []byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []byte
		wantErr error
	}{
		{
			name:    "not exists",
			fields:  fields{data: map[string][]byte{}},
			args:    args{key: "test", data: []byte("good")},
			want:    []byte("good"),
			wantErr: nil,
		},
		{
			name:    "exists",
			fields:  fields{data: map[string][]byte{"test": []byte("bad")}},
			args:    args{key: "test", data: []byte("good")},
			want:    []byte("good"),
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewMemCache().(*memCache)
			for key, val := range tt.fields.data {
				m.data[key] = val
			}
			if err := m.Put(context.TODO(), tt.args.key, tt.args.data); err != tt.wantErr {
				t.Errorf("Put() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got := m.data[tt.args.key]; !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Get() got = %v, want %v", got, tt.want)
			}
		})
	}
}
