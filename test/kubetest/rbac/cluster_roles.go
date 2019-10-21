// Copyright 2019 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0
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

package rbac

import (
	"context"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type Role interface {
	Create(kubernetes.Interface) error
	Delete(kubernetes.Interface, string) error
	GetName() string
	//Wait - workaround for https://github.com/kubernetes/kubernetes/issues/66689
	Wait(context.Context, kubernetes.Interface) error
}

type ClusterRole struct {
	rbacv1.ClusterRole
}

func (r *ClusterRole) Create(clientset kubernetes.Interface) error {
	_, err := clientset.RbacV1().ClusterRoles().Create(&r.ClusterRole)
	return err
}

func (r *ClusterRole) Delete(clientset kubernetes.Interface, name string) error {
	return clientset.RbacV1().ClusterRoles().Delete(name, &metav1.DeleteOptions{})
}

func (r *ClusterRole) GetName() string {
	return r.ObjectMeta.Name
}

func (r *ClusterRole) Wait(ctx context.Context, client kubernetes.Interface) error {
	return waitFor(ctx, func() bool {
		_, err := client.RbacV1().ClusterRoles().Get(r.Name, metav1.GetOptions{})
		if err != nil {
			logrus.Infof("An error during get cluster role: %v", err)
			return false
		}
		return true
	})
}

type ClusterRoleBinding struct {
	rbacv1.ClusterRoleBinding
}

func (r *ClusterRoleBinding) Wait(ctx context.Context, client kubernetes.Interface) error {
	return waitFor(ctx, func() bool {
		role, err := client.RbacV1().ClusterRoleBindings().Get(r.Name, metav1.GetOptions{})
		if err != nil {
			logrus.Infof("An error during get clustr role bindings: %v", err)
			return false
		}
		if len(role.Subjects) != len(r.Subjects) {
			logrus.Infof("Incomming role binding has incorrect size, %v != %v", r.ClusterRoleBinding, role)
			return false
		}
		subject := &role.Subjects[0]
		if subject.Namespace != r.Subjects[0].Namespace {
			logrus.Infof("Incomming role binding has wrong subject, %v != %v", r.ClusterRoleBinding.Subjects, role.Subjects)
			return false
		}

		if _, err = client.CoreV1().ServiceAccounts(subject.Namespace).Get(subject.Name, metav1.GetOptions{}); err != nil {
			logrus.Infof("Service account not created, err: %v", err)
			return false
		}
		return true
	})
}
func (r *ClusterRoleBinding) Create(clientset kubernetes.Interface) error {
	_, err := clientset.RbacV1().ClusterRoleBindings().Create(&r.ClusterRoleBinding)
	return err
}

func (r *ClusterRoleBinding) Delete(clientset kubernetes.Interface, name string) error {
	return clientset.RbacV1().ClusterRoleBindings().Delete(name, &metav1.DeleteOptions{})
}

func (r *ClusterRoleBinding) GetName() string {
	return r.ObjectMeta.Name
}

/**
Roles is a map containing simplified roles names for external usage and mapping them to a function
that creates object of the required type.
*/
var Roles = map[string]func(string, string) Role{
	"admin":   CreateRoleAdmin,
	"view":    CreateRoleView,
	"binding": CreateRoleBinding,
}

/**
RoleNames is a map where the keys are simplified roles names for external usage and the values are
the real roles names in the Kubernetes cluster
*/
var RoleNames = map[string]string{
	"admin":   "nsm-role",
	"view":    "aggregate-network-services-view",
	"binding": "nsm-role-binding",
}

// CreateRoleAdmin creates a role with admin permissions.
func CreateRoleAdmin(name, namespace string) Role {
	roleAdmin := &ClusterRole{
		ClusterRole: rbacv1.ClusterRole{
			TypeMeta: metav1.TypeMeta{
				Kind: "ClusterRole",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
				Labels: map[string]string{
					"rbac.authorization.k8s.io/aggregate-to-admin": "true",
					"rbac.authorization.k8s.io/aggregate-to-edit":  "true",
				},
				Namespace: namespace,
			},
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{"networkservicemesh.io"},
					Resources: []string{
						"networkservices",
						"networkserviceendpoints",
						"networkservicemanagers",
					},
					Verbs: []string{"*"},
				},
				{
					APIGroups: []string{"apiextensions.k8s.io"},
					Resources: []string{"customresourcedefinitions"},
					Verbs:     []string{"*"},
				},
				{
					APIGroups: []string{""},
					Resources: []string{"configmaps"},
					Verbs:     []string{"get"},
				},
				{
					APIGroups: []string{""},
					Resources: []string{"nodes", "services", "namespaces"},
					Verbs:     []string{"get", "list", "watch"},
				},
			},
		},
	}
	return roleAdmin
}

// CreateRoleView creates a role with view permissions
func CreateRoleView(name, namespace string) Role {
	roleView := &ClusterRole{
		ClusterRole: rbacv1.ClusterRole{
			TypeMeta: metav1.TypeMeta{
				Kind: "ClusterRole",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
				Labels: map[string]string{
					"rbac.authorization.k8s.io/aggregate-to-view": "true",
				},
				Namespace: namespace,
			},
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups: []string{"networkservicemesh.io"},
					Resources: []string{"networkservices"},
					Verbs:     []string{"get", "list", "watch"},
				},
			},
		},
	}
	return roleView
}

// CreateRoleBinding creates a role with binding permissions
func CreateRoleBinding(name, namespace string) Role {
	roleBinding := &ClusterRoleBinding{
		ClusterRoleBinding: rbacv1.ClusterRoleBinding{
			TypeMeta: metav1.TypeMeta{
				Kind: "ClusterRoleBinding",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "ClusterRole",
				Name:     "nsm-role",
			},
			Subjects: []rbacv1.Subject{
				{
					Kind:      "ServiceAccount",
					APIGroup:  "",
					Name:      "nsmgr-acc",
					Namespace: namespace,
				},
			},
		},
	}
	return roleBinding
}

func DeleteAllRoles(clientset kubernetes.Interface) error {
	if err := (&ClusterRole{}).Delete(clientset, RoleNames["admin"]); err != nil {
		logrus.Error("Unable to delete kubernetes admin roles, aborting", err)
		return err
	}
	if err := (&ClusterRole{}).Delete(clientset, RoleNames["view"]); err != nil {
		logrus.Error("Unable to delete kubernetes view roles, aborting", err)
		return err
	}
	if err := (&ClusterRoleBinding{}).Delete(clientset, RoleNames["binding"]); err != nil {
		logrus.Error("Unable to delete kubernetes binding roles, aborting", err)
		return err
	}

	return nil
}

func waitFor(ctx context.Context, exit func() bool) error {
	wg := sync.WaitGroup{}
	wg.Add(1)
	var err error
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				err = ctx.Err()
				return
			default:
				if exit() {
					return
				}
				<-time.After(time.Millisecond * 200)
			}
		}
	}()
	wg.Wait()
	return err
}
