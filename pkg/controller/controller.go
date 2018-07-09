package controller

import (
	"fmt"
	"log"
	"sync"
	"time"

	"k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

// NamespaceController watches the kubernetes api for changes to namespaces and
// creates a Rolebinding for that particular namespace.
type NamespaceController struct {
	namespaceInformer cache.SharedIndexInformer
	kclient *kubernetes.Clientset
}

// Run starts the process for listening for namespace changes and acting upon those changes.
func (c *NamespaceController) Run(stopCh <-chan struct{}, wg *sync.WaitGroup) {

	// When this function completes, mark the go function as done
	defer wg.Done()

	// Increment wait group before creating a new goroutine
	wg.Add(1)

	// Execute goroutine
	go c.namespaceInformer.Run(stopCh)

	// Wait until we receive a stop signal
	<-stopCh
}

//NewNamespaceController creates a new NamespaceController
func NewNamespaceController(kclient *kubernetes.Clientset) *NamespaceController {

	namespaceController := &NamespaceController{}

	// Create informer for watching Namespaces
	namespaceInformer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				return kclient.CoreV1().Namespaces().List(options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return kclient.CoreV1().Namespaces().Watch(options)
			},
		},
			&v1.Namespace{},
			3*time.Minute,
			cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc},
	)


	namespaceInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: namespaceController.createRoleBinding,
	})

	namespaceController.kclient = kclient
	namespaceController.namespaceInformer = namespaceInformer

	return namespaceController
}

func (c *NamespaceController) createRoleBinding(obj interface{}) {
	namespaceObj := obj.(*v1.Namespace)
	namespaceName := namespaceObj.Name

	roleBinding := &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "RoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("ad-kubernetes-%s", namespaceName),
			Namespace: namespaceName,
		},
		Subjects: []rbacv1.Subject{
			rbacv1.Subject{
				Kind: "Group",
				Name: fmt.Sprintf("ad-kubernetes-%s", namespaceName),
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "edit",
		},
	}

	_, err := c.kclient.RbacV1().RoleBindings(namespaceName).Create(roleBinding)

	if err != nil {
		log.Println(fmt.Sprintf("Failed to create Role Binding: %s", err.Error()))
	} else {
		log.Println(fmt.Sprintf("Created AD RoleBinding for Namespace: %s", roleBinding.Name))
	}
}