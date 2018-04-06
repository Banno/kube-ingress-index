// Copyright 2018 Jack Henry and Associates
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// kube-inress-index is a   // TODO(adam)
//
// https://github.com/kubernetes/client-go/tree/master/examples
// https://github.com/jetstack/kube-lego
//
package main

import (
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"time"

	cache "k8s.io/client-go/tools/cache"
	clientcmd "k8s.io/client-go/tools/clientcmd"
	k8sExtensions "k8s.io/client-go/pkg/apis/extensions/v1beta1"
	k8sMeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubernetes "k8s.io/client-go/kubernetes"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	// workqueue "k8s.io/client-go/util/workqueue"

	// Import OIDC provider -- https://github.com/coreos/tectonic-forum/issues/99
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
)

var (
	resyncInterval = 60 * time.Second
)

func main() {
	var kubeconfig *string
	if home := homeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	// TODO(adam)
	// "k8s.io/client-go/rest"
	// config, err := rest.InClusterConfig()
	// kubeClient, err := kubernetes.NewForConfig(config)

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	// ingress
	respChan := make(chan []ingress, 10)
	go watchIngresses(clientset, []string{
		"infrastructure",
	}, respChan)

	// setup http page, forever blocks
	listenHttp(":8080", respChan)
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}

func listenHttp(address string, respChan chan []ingress) {
	var curIngresses []ingress

	go func() {
		for {
			select{
			case cur := <-respChan:
				curIngresses = cur
				fmt.Printf("got %d ingresses\n", len(cur))
			}
		}
	}()

	handler := func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "%d ingresses", len(curIngresses))
	}
	http.HandleFunc("/", handler)
	http.ListenAndServe(address, nil)
}

func ingressListFunc(c *kubernetes.Clientset, ns string) func(k8sMeta.ListOptions) (runtime.Object, error) {
	return func(opts k8sMeta.ListOptions) (runtime.Object, error) {
		return c.Extensions().Ingresses(ns).List(opts)
	}
}

func ingressWatchFunc(c *kubernetes.Clientset, ns string) func(options k8sMeta.ListOptions) (watch.Interface, error) {
	return func(options k8sMeta.ListOptions) (watch.Interface, error) {
		return c.Extensions().Ingresses(ns).Watch(options)
	}
}

// TODO(adam): return multiple FQDN's
func buildFQDN(ing k8sExtensions.IngressSpec) string {
	// TODO(adam): we assume TLS for now, but should lookup in ing.TLS https://godoc.org/k8s.io/api/extensions/v1beta1#IngressTLS
	for i := range ing.Rules {
		// find a rule with a host and path that's parsable // TODO
		host := ing.Rules[i].Host
		paths := ing.Rules[i].IngressRuleValue.HTTP.Paths
		for i := range paths {
			u, _ := url.Parse(fmt.Sprintf("https://%s", host))
			if u == nil {
				continue
			}
			u.Path = paths[i].Path
			return u.String()
		}
	}
	return ""
}

// ingress is a smaller model for internal shipping about
type ingress struct {
	Name, Namespace string

	// FQDN is an address which the backend is reachable from
	FQDN string
}
func (ing ingress) String() string {
	return fmt.Sprintf("Ingress: namespace=%s, name=%s, fqdn=%s", ing.Namespace, ing.Name, ing.FQDN)
}

type ingresses struct {
	// current set of Ingress objects
	active []ingress
	mu sync.Mutex
}
func (i *ingresses) upsert(ing ingress) []ingress {
	i.mu.Lock()
	defer i.mu.Unlock()

	for k := range i.active {
		if i.active[k].FQDN == ing.FQDN {
			var out []ingress
			copy(out, i.active)
			return out // return early, we've already added this ingress
		}
	}
	// didn't find our ingress, add it and return
	i.active = append(i.active, ing)

	// return a copy
	out := make([]ingress, len(i.active))
	fmt.Printf("copied %d\n", copy(out, i.active))
	return out
}
func (i *ingresses) delete(ing ingress) []ingress {
	i.mu.Lock()
	defer i.mu.Unlock()

	var next []ingress
	for k := range i.active {
		if i.active[k].FQDN == ing.FQDN {
			continue
		}
		next = append(next, i.active[k])
	}
	i.active = next

	// return a copy
	out := make([]ingress, len(i.active))
	copy(out, i.active)
	return out
}

func watchIngresses(kubeClient *kubernetes.Clientset, namespaces []string, respChan chan []ingress) {
	// Internal accumulator, a copy is sent back each time
	accum := &ingresses{}

	ingEventHandler := cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			addIng, ok := obj.(*k8sExtensions.Ingress)
			if ok {
				ing := ingress{
					Namespace: addIng.Namespace,
					Name: addIng.Name,
					FQDN: buildFQDN(addIng.Spec),
				}
				current := accum.upsert(ing)
				respChan <- current
				fmt.Printf("added %s, len(current)=%d\n", ing.String(), len(current))
			}
			// kl.workQueue.Add(true) // TODO(adam): uhh...
		},
		DeleteFunc: func(obj interface{}) {
			delIng, ok := obj.(*k8sExtensions.Ingress)
			if ok {
				ing := ingress{
					Namespace: delIng.Namespace,
					Name: delIng.Name,
					FQDN: buildFQDN(delIng.Spec),
				}
				current := accum.delete(ing)
				respChan <- current
				fmt.Printf("deleted %s\n", ing.String())
			}
		},
		UpdateFunc: func(_, cur interface{}) {
			upIng, ok := cur.(*k8sExtensions.Ingress)
			if ok {
				ing := ingress{
					Namespace: upIng.Namespace,
					Name: upIng.Name,
					FQDN: buildFQDN(upIng.Spec),
				}
				current := accum.upsert(ing)
				respChan <- current
				fmt.Printf("updated %s\n", ing.String())
			}
		},
	}

	_, controller := cache.NewInformer(
		&cache.ListWatch{
			ListFunc:  ingressListFunc(kubeClient, namespaces[0]), // TODO(adam): actually process all namespaces
			WatchFunc: ingressWatchFunc(kubeClient, namespaces[0]),
		},
		&k8sExtensions.Ingress{},
		resyncInterval,
		ingEventHandler,
	)

	controller.Run(nil) // TODO(adam): watch for signals and shutdown
}
