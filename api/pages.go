package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

// ListPagesOptions contains options for listing pages.
type ListPagesOptions struct {
	Limit    int
	Cursor   string
	Status   string // current, archived, draft
	Sort     string // title, -title, created-date, -created-date, modified-date, -modified-date
	Title    string // Filter by title (contains)
	BodyFormat string // storage, atlas_doc_format, view
}

// GetPageOptions contains options for getting a page.
type GetPageOptions struct {
	BodyFormat string // storage, atlas_doc_format, view
}

// ListPages returns a list of pages in a space.
func (c *Client) ListPages(ctx context.Context, spaceID string, opts *ListPagesOptions) (*PaginatedResponse[Page], error) {
	params := url.Values{}
	params.Set("limit", "25") // Default limit

	if opts != nil {
		if opts.Limit > 0 {
			params.Set("limit", strconv.Itoa(opts.Limit))
		}
		if opts.Cursor != "" {
			params.Set("cursor", opts.Cursor)
		}
		if opts.Status != "" {
			params.Set("status", opts.Status)
		}
		if opts.Sort != "" {
			params.Set("sort", opts.Sort)
		}
		if opts.Title != "" {
			params.Set("title", opts.Title)
		}
		if opts.BodyFormat != "" {
			params.Set("body-format", opts.BodyFormat)
		}
	}

	path := fmt.Sprintf("/api/v2/spaces/%s/pages?%s", spaceID, params.Encode())
	body, err := c.Get(ctx, path)
	if err != nil {
		return nil, err
	}

	var result PaginatedResponse[Page]
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse pages response: %w", err)
	}

	return &result, nil
}

// GetPage returns a single page by ID.
func (c *Client) GetPage(ctx context.Context, pageID string, opts *GetPageOptions) (*Page, error) {
	params := url.Values{}
	if opts != nil && opts.BodyFormat != "" {
		params.Set("body-format", opts.BodyFormat)
	}

	path := fmt.Sprintf("/api/v2/pages/%s", pageID)
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	body, err := c.Get(ctx, path)
	if err != nil {
		return nil, err
	}

	var page Page
	if err := json.Unmarshal(body, &page); err != nil {
		return nil, fmt.Errorf("failed to parse page response: %w", err)
	}

	return &page, nil
}

// CreatePage creates a new page.
func (c *Client) CreatePage(ctx context.Context, req *CreatePageRequest) (*Page, error) {
	body, err := c.Post(ctx, "/api/v2/pages", req)
	if err != nil {
		return nil, err
	}

	var page Page
	if err := json.Unmarshal(body, &page); err != nil {
		return nil, fmt.Errorf("failed to parse create page response: %w", err)
	}

	return &page, nil
}

// UpdatePage updates an existing page.
func (c *Client) UpdatePage(ctx context.Context, pageID string, req *UpdatePageRequest) (*Page, error) {
	path := fmt.Sprintf("/api/v2/pages/%s", pageID)
	body, err := c.Put(ctx, path, req)
	if err != nil {
		return nil, err
	}

	var page Page
	if err := json.Unmarshal(body, &page); err != nil {
		return nil, fmt.Errorf("failed to parse update page response: %w", err)
	}

	return &page, nil
}

// DeletePage deletes a page.
func (c *Client) DeletePage(ctx context.Context, pageID string) error {
	path := fmt.Sprintf("/api/v2/pages/%s", pageID)
	_, err := c.Delete(ctx, path)
	return err
}
