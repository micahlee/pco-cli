package api

import (
	"context"
	"iter"
	"net/url"

	"github.com/micahlee/pco-cli/internal/models"
)

// PageIterator returns an iterator that lazily fetches all pages for a list endpoint.
// It follows links.next until exhausted.
func (c *Client) PageIterator(ctx context.Context, path string, params url.Values) iter.Seq2[models.Resource, error] {
	return func(yield func(models.Resource, error) bool) {
		// Build initial URL with params
		currentPath := path
		if len(params) > 0 {
			currentPath += "?" + params.Encode()
		}

		for currentPath != "" {
			data, err := c.do(ctx, "GET", currentPath, "")
			if err != nil {
				yield(models.Resource{}, err)
				return
			}

			resources, links, err := models.ParseList(data)
			if err != nil {
				yield(models.Resource{}, err)
				return
			}

			for _, r := range resources {
				if !yield(r, nil) {
					return
				}
			}

			// Follow next link
			if links != nil && links.Next != "" {
				currentPath = links.Next
			} else {
				currentPath = ""
			}
		}
	}
}

// GetAll fetches all resources from a paginated endpoint into a slice.
func (c *Client) GetAll(ctx context.Context, path string, params url.Values) ([]models.Resource, error) {
	var all []models.Resource
	for r, err := range c.PageIterator(ctx, path, params) {
		if err != nil {
			return nil, err
		}
		all = append(all, r)
	}
	return all, nil
}
