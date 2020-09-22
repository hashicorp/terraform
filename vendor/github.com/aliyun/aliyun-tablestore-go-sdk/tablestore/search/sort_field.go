package search

import "github.com/aliyun/aliyun-tablestore-go-sdk/tablestore/otsprotocol"

type NestedFilter struct {
	Path   string
	Filter Query
}

func (f *NestedFilter) ProtoBuffer() (*otsprotocol.NestedFilter, error) {
	pbF := &otsprotocol.NestedFilter{
		Path: &f.Path,
	}
	pbQ, err := f.Filter.ProtoBuffer()
	if err != nil {
		return nil, err
	}
	pbF.Filter = pbQ
	return pbF, err
}

type FieldSort struct {
	FieldName    string
	Order        *SortOrder
	Mode         *SortMode
	NestedFilter *NestedFilter
}

func NewFieldSort(fieldName string, order SortOrder) *FieldSort {
	return &FieldSort{
		FieldName: fieldName,
		Order:     order.Enum(),
	}
}

func (s *FieldSort) ProtoBuffer() (*otsprotocol.Sorter, error) {
	pbFieldSort := &otsprotocol.FieldSort{
		FieldName: &s.FieldName,
	}
	if s.Order != nil {
		pbOrder, err := s.Order.ProtoBuffer()
		if err != nil {
			return nil, err
		}
		pbFieldSort.Order = pbOrder
	}
	if s.Mode != nil {
		pbMode, err := s.Mode.ProtoBuffer()
		if err != nil {
			return nil, err
		}
		if pbMode != nil {
			pbFieldSort.Mode = pbMode
		}
	}
	if s.NestedFilter != nil {
		pbFilter, err := s.NestedFilter.ProtoBuffer()
		if err != nil {
			return nil, err
		}
		pbFieldSort.NestedFilter = pbFilter
	}
	pbSorter := &otsprotocol.Sorter{
		FieldSort: pbFieldSort,
	}
	return pbSorter, nil
}
