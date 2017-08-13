// Package implements OCCM Working Environments API (VSA)
package vsa

import (
  "github.com/candidpartners/occm-sdk-go/api/workenv"
)

type VSAWorkingEnvironmentAPIProto interface {
	GetAggregates(string) ([]workenv.AggregateResponse, error)
  GetVolumes(string) ([]workenv.VolumeResponse, error)
  QuoteVolume(*VSAVolumeQuoteRequest) (*VSAVolumeQuoteResponse, error)
  CreateVolume(bool, *VSAVolumeCreateRequest) (string, error)
  ModifyVolume(string, string, string, *workenv.VolumeModifyRequest) (string, error)
  DeleteVolume(string, string, string) (string, error)
  MoveVolume(string, string, string, *workenv.VolumeMoveRequest) (string, error)
  CloneVolume(string, string, string, *workenv.VolumeCloneRequest) (string, error)
  ChangeVolumeTier(string, string, string, *workenv.ChangeVolumeTierRequest) (string, error)
}

var _ VSAWorkingEnvironmentAPIProto = (*VSAWorkingEnvironmentAPI)(nil)
