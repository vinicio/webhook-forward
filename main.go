package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi"
	"github.com/google/go-github/github"
)

// b:master,l:team/backend=webhook-do-backend.com&b:frontend,l:team/backend=webhook-do-frontend.com

func main() {
	r := chi.NewRouter()

	r.Post("/webhook", webhook)

	log.Print("Listening on :9090")
	if err := http.ListenAndServe(":9090", r); err != nil {
		log.Fatalf("could not listen: %v", err)
	}
}

func webhook(w http.ResponseWriter, r *http.Request) {
	branches := map[string]string{}
	labels := map[string]string{}

	for rules, hook := range r.URL.Query() {
		if len(hook) > 0 {
			for _, rule := range strings.Split(rules, ",") {
				if strings.HasPrefix(rule, "l:") {
					labels[rule[2:]] = hook[0]
				}

				if strings.HasPrefix(rule, "b:") {
					branches[rule[2:]] = hook[0]
				}
			}
		}
	}

	payload, err := github.ValidatePayload(r, nil)
	if err != nil {
		log.Print(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	event, err := github.ParseWebHook(github.WebHookType(r), payload)
	if err != nil {
		log.Print(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	switch event.(type) {
	case *github.PushEvent:
		for branch, webhook := range branches {
			if *event.(*github.PushEvent).BaseRef == fmt.Sprintf("refs/heads/%s", branch) {
				forward(w, r, webhook)
				w.WriteHeader(http.StatusNoContent)
				return
			}
		}

		forward(w, r, onlyValues(branches)...)

	case *github.PullRequestEvent:
		branch := event.(*github.PullRequestEvent).PullRequest.Base.Ref
		if branch == nil {
			log.Print("PullRequestEvent: nil Base.Ref")
			forward(w, r, onlyValues(branches)...)
			return
		}

		webhook, ok := branches[*branch]
		if !ok {
			forward(w, r, onlyValues(branches)...)
			return
		}

		forward(w, r, webhook)

	case *github.PullRequestReviewEvent:
		branch := event.(*github.PullRequestReviewEvent).PullRequest.Base.Ref
		if branch == nil {
			log.Print("PullRequestEvent: nil Base.Ref")
			forward(w, r, onlyValues(branches)...)
			return
		}

		webhook, ok := branches[*branch]
		if !ok {
			forward(w, r, onlyValues(branches)...)
			return
		}

		forward(w, r, webhook)

	case *github.PullRequestReviewCommentEvent:
		branch := event.(*github.PullRequestReviewCommentEvent).PullRequest.Base.Ref
		if branch == nil {
			log.Print("PullRequestEvent: nil Base.Ref")
			forward(w, r, onlyValues(branches)...)
			return
		}

		webhook, ok := branches[*branch]
		if !ok {
			forward(w, r, onlyValues(branches)...)
			return
		}

		forward(w, r, webhook)

	case *github.IssuesEvent:
		for _, label := range event.(*github.IssuesEvent).Issue.Labels {
			if webhook, ok := labels[*label.Name]; ok {
				forward(w, r, webhook)
				return
			}
		}

		forward(w, r, onlyValues(labels)...)

	case *github.IssueEvent:
		for _, label := range event.(*github.IssueEvent).Issue.Labels {
			if webhook, ok := labels[*label.Name]; ok {
				forward(w, r, webhook)
				return
			}
		}

		forward(w, r, onlyValues(labels)...)

	case *github.IssueCommentEvent:
		for _, label := range event.(*github.IssueCommentEvent).Issue.Labels {
			if webhook, ok := labels[*label.Name]; ok {
				forward(w, r, webhook)
				return
			}
		}

		forward(w, r, onlyValues(labels)...)

	default:
		forward(w, r, append(onlyValues(labels), onlyValues(branches)...)...)
	}
}

func forward(w http.ResponseWriter, r *http.Request, destinations ...string) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	for _, url := range destinations {
		proxy, err := http.NewRequest(r.Method, url, bytes.NewReader(body))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		proxy.Header = r.Header

		resp, err := http.DefaultClient.Do(proxy)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode >= http.StatusMultipleChoices {
			http.Error(w, resp.Status, resp.StatusCode)
			return
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

func onlyValues(input map[string]string) []string {
	values := make([]string, len(input))

	for _, v := range input {
		values = append(values, v)
	}

	return values
}
