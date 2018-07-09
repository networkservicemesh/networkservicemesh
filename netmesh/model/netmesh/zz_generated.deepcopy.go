// +build !ignore_autogenerated

// Copyright (c) 2018 Cisco and/or its affiliates.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Code generated by deepcopy-gen. DO NOT EDIT.

package netmesh

import (
	common "github.com/ligato/networkservicemesh/pkg/nsm/apis/common"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NetworkService) DeepCopyInto(out *NetworkService) {
	*out = *in
	if in.Metadata != nil {
		in, out := &in.Metadata, &out.Metadata
		*out = new(common.Metadata)
		(*in).DeepCopyInto(*out)
	}
	if in.Channel != nil {
		in, out := &in.Channel, &out.Channel
		*out = make([]*NetworkServiceChannel, len(*in))
		for i := range *in {
			if (*in)[i] != nil {
				in, out := &(*in)[i], &(*out)[i]
				*out = new(NetworkServiceChannel)
				(*in).DeepCopyInto(*out)
			}
		}
	}
	out.XXX_NoUnkeyedLiteral = in.XXX_NoUnkeyedLiteral
	if in.XXX_unrecognized != nil {
		in, out := &in.XXX_unrecognized, &out.XXX_unrecognized
		*out = make([]byte, len(*in))
		copy(*out, *in)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NetworkService.
func (in *NetworkService) DeepCopy() *NetworkService {
	if in == nil {
		return nil
	}
	out := new(NetworkService)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NetworkServiceChannel) DeepCopyInto(out *NetworkServiceChannel) {
	*out = *in
	if in.Metadata != nil {
		in, out := &in.Metadata, &out.Metadata
		*out = new(common.Metadata)
		(*in).DeepCopyInto(*out)
	}
	if in.Interface != nil {
		in, out := &in.Interface, &out.Interface
		*out = make([]*common.Interface, len(*in))
		for i := range *in {
			if (*in)[i] != nil {
				in, out := &(*in)[i], &(*out)[i]
				*out = new(common.Interface)
				(*in).DeepCopyInto(*out)
			}
		}
	}
	out.XXX_NoUnkeyedLiteral = in.XXX_NoUnkeyedLiteral
	if in.XXX_unrecognized != nil {
		in, out := &in.XXX_unrecognized, &out.XXX_unrecognized
		*out = make([]byte, len(*in))
		copy(*out, *in)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NetworkServiceChannel.
func (in *NetworkServiceChannel) DeepCopy() *NetworkServiceChannel {
	if in == nil {
		return nil
	}
	out := new(NetworkServiceChannel)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NetworkServiceEndpoint) DeepCopyInto(out *NetworkServiceEndpoint) {
	*out = *in
	if in.Metadata != nil {
		in, out := &in.Metadata, &out.Metadata
		*out = new(common.Metadata)
		(*in).DeepCopyInto(*out)
	}
	out.XXX_NoUnkeyedLiteral = in.XXX_NoUnkeyedLiteral
	if in.XXX_unrecognized != nil {
		in, out := &in.XXX_unrecognized, &out.XXX_unrecognized
		*out = make([]byte, len(*in))
		copy(*out, *in)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NetworkServiceEndpoint.
func (in *NetworkServiceEndpoint) DeepCopy() *NetworkServiceEndpoint {
	if in == nil {
		return nil
	}
	out := new(NetworkServiceEndpoint)
	in.DeepCopyInto(out)
	return out
}
