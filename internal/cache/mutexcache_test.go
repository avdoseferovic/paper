package cache_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/avdoseferovic/paper/internal/cache"

	"github.com/avdoseferovic/paper/internal/mocks"
	"github.com/avdoseferovic/paper/pkg/consts/extension"
	"github.com/avdoseferovic/paper/pkg/core/entity"

	"github.com/avdoseferovic/paper/internal/assert"
)

func TestNewMutexDecorator(t *testing.T) {
	t.Parallel()
	// Act
	sut := cache.NewMutexDecorator(nil)

	// Assert
	assert.NotNil(t, sut)
	assert.Equal(t, "*cache.mutexCache", fmt.Sprintf("%T", sut))
}

func TestMutexCache_AddImage(t *testing.T) {
	t.Parallel()
	// Arrange
	value := "value1"
	img := &entity.Image{}

	innerMock := mocks.NewCache(t)
	innerMock.EXPECT().AddImage(value, img).Once()

	sut := cache.NewMutexDecorator(innerMock)

	// Act
	sut.AddImage(value, img)
}

func TestMutexCache_GetImage(t *testing.T) {
	t.Parallel()
	// Arrange
	value := "value2"
	ext := extension.Jpg
	imgToReturn := &entity.Image{}
	errToReturn := errors.New("any error")

	innerMock := mocks.NewCache(t)
	innerMock.EXPECT().GetImage(value, ext).Return(imgToReturn, errToReturn).Once()

	sut := cache.NewMutexDecorator(innerMock)

	// Act
	img, err := sut.GetImage(value, ext)

	// Assert
	assert.Equal(t, imgToReturn, img)
	assert.Equal(t, errToReturn, err)
}

func TestMutexCache_LoadImage(t *testing.T) {
	t.Parallel()
	// Arrange
	value := "value3"
	ext := extension.Jpg
	errToReturn := errors.New("any error")

	innerMock := mocks.NewCache(t)
	innerMock.EXPECT().LoadImage(value, ext).Return(errToReturn).Once()

	sut := cache.NewMutexDecorator(innerMock)

	// Act
	err := sut.LoadImage(value, ext)

	// Assert
	assert.Equal(t, errToReturn, err)
}
