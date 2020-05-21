package forward

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/google/go-github/github"
)

func WebhookForward(w http.ResponseWriter, r *http.Request) {
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

	payload, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("read request body: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Restores request body
	r.Body.Close()
	r.Body = ioutil.NopCloser(bytes.NewBuffer(payload))

	event, err := github.ParseWebHook(github.WebHookType(r), payload)
	if err != nil {
		log.Printf("parse webhook: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	switch event.(type) {
	case *github.PushEvent:
		if event.(*github.PushEvent).Ref == nil {
			forward(w, r, onlyValues(branches)...)
			return
		}

		for branch, webhook := range branches {
			if *event.(*github.PushEvent).Ref == fmt.Sprintf("refs/heads/%s", branch) {
				forward(w, r, webhook)
				return
			}
		}

		forward(w, r, onlyValues(branches)...)

	case *github.PullRequestEvent:
		branch := event.(*github.PullRequestEvent).PullRequest.Base.Ref
		if branch == nil {
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
		e := event.(*github.PullRequestReviewEvent)
		if e.PullRequest == nil || e.PullRequest.Base == nil || e.PullRequest.Base.Ref == nil {
			forward(w, r, onlyValues(branches)...)
			return
		}

		branch := e.PullRequest.Base.Ref

		webhook, ok := branches[*branch]
		if !ok {
			forward(w, r, onlyValues(branches)...)
			return
		}

		forward(w, r, webhook)

	case *github.PullRequestReviewCommentEvent:
		e := event.(*github.PullRequestReviewCommentEvent)
		if e.PullRequest == nil || e.PullRequest.Base == nil || e.PullRequest.Base.Ref == nil {
			forward(w, r, onlyValues(branches)...)
			return
		}

		branch := e.PullRequest.Base.Ref

		webhook, ok := branches[*branch]
		if !ok {
			forward(w, r, onlyValues(branches)...)
			return
		}

		forward(w, r, webhook)

	case *github.IssuesEvent:
		e := event.(*github.IssuesEvent)
		if e.Issue == nil {
			forward(w, r, onlyValues(labels)...)
			return
		}

		for _, label := range e.Issue.Labels {
			if webhook, ok := labels[*label.Name]; ok {
				forward(w, r, webhook)
				return
			}
		}

		forward(w, r, onlyValues(labels)...)

	case *github.IssueEvent:
		e := event.(*github.IssueEvent)
		if e.Issue == nil {
			forward(w, r, onlyValues(labels)...)
			return
		}

		for _, label := range e.Issue.Labels {
			if webhook, ok := labels[*label.Name]; ok {
				forward(w, r, webhook)
				return
			}
		}

		forward(w, r, onlyValues(labels)...)

	case *github.IssueCommentEvent:
		e := event.(*github.IssueCommentEvent)
		if e.Issue == nil {
			forward(w, r, onlyValues(labels)...)
			return
		}

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
	values := make([]string, 0, len(input))

	for _, v := range input {
		values = append(values, v)
	}

	return values
}
