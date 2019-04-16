package rbac

import (
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type Role interface {
	Create(kubernetes.Interface) error
	Delete(kubernetes.Interface, string) error
	GetName() string
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

type ClusterRoleBinding struct {
	rbacv1.ClusterRoleBinding
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
var Roles = map[string]func(string) Role{
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

func CreateRoleAdmin(name string) Role {
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
			},
		},
	}
	return roleAdmin
}

func CreateRoleView(name string) Role {
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

func CreateRoleBinding(name string) Role {
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
					Name:      "default",
					Namespace: "default",
				},
			},
		},
	}
	return roleBinding
}
