package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/example/outbox-practice/core/internal/domain"
	"github.com/example/outbox-practice/core/internal/repository"
	"github.com/example/outbox-practice/internal/event"
)

type PostService struct {
	repo           *repository.Repository
	topic          string
	idempotencyTTL time.Duration
}

func NewPostService(repo *repository.Repository, topic string, ttl time.Duration) *PostService {
	return &PostService{repo: repo, topic: topic, idempotencyTTL: ttl}
}

func (s *PostService) Create(ctx context.Context, key, title, body string) (domain.Post, bool, error) {
	now := time.Now().UTC()
	p := domain.Post{ID: uuid.NewString(), Title: strings.TrimSpace(title), Body: strings.TrimSpace(body), CreatedAt: now}
	e := event.Envelope[event.PostCreated]{ID: uuid.NewString(), Type: event.PostCreatedV1, AggregateID: p.ID, AggregateVersion: 1, OccurredAt: now, Data: event.PostCreated{Title: p.Title, Body: p.Body}}
	payload, err := json.Marshal(e)
	if err != nil {
		return domain.Post{}, false, err
	}
	canonicalRequest, err := json.Marshal(struct {
		Title string `json:"title"`
		Body  string `json:"body"`
	}{Title: p.Title, Body: p.Body})
	if err != nil {
		return domain.Post{}, false, err
	}
	h := sha256.Sum256(canonicalRequest)
	return s.repo.CreatePost(ctx, repository.CreatePostParams{Post: p, EventID: e.ID, Topic: s.topic, Payload: payload, IdempotencyKey: key, RequestHash: hex.EncodeToString(h[:]), ExpiresAt: now.Add(s.idempotencyTTL)})
}

func (s *PostService) List(ctx context.Context) ([]domain.Post, error) { return s.repo.ListPosts(ctx) }
