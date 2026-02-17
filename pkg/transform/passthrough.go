package transform

import (
	"github.com/IEatCodeDaily/data-pipe/pkg/pipeline"
)

// PassThroughTransformer is a transformer that passes events through unchanged
type PassThroughTransformer struct{}

// NewPassThroughTransformer creates a new pass-through transformer
func NewPassThroughTransformer() *PassThroughTransformer {
	return &PassThroughTransformer{}
}

// Transform passes the event through unchanged
func (t *PassThroughTransformer) Transform(event pipeline.Event) (pipeline.Event, error) {
	return event, nil
}
