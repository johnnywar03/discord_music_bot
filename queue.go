package main

import (
	"errors"
	"fmt"
)

type VideoQueue struct {
	CurrentVideo *VideoNode
}

type VideoNode struct {
	Id        string
	Title     string
	NextVideo *VideoNode
}

func (queue *VideoQueue) add(id string) (title string, err error) {
	// Get video title
	title = getTitle(id)
	if title == "" {
		return "", errors.New("Cannot get video title.")
	}

	newVideo := &VideoNode{
		Id:        id,
		Title:     title,
		NextVideo: nil,
	}

	// Add the first video in the queue
	if queue.CurrentVideo == nil {
		queue.CurrentVideo = newVideo
		return queue.CurrentVideo.Title, nil
	} else {
		// Init queue for loop through all video info
		loopQueue := queue.CurrentVideo

		// Next video until the end
		for loopQueue.NextVideo != nil {
			loopQueue = loopQueue.NextVideo
		}
		// Add new video at the end
		loopQueue.NextVideo = newVideo
		return loopQueue.NextVideo.Title, nil
	}
}

func (queue *VideoQueue) deleteFirst() {
	if queue.CurrentVideo == nil {
		return
	}
	// Replace first video with the second video
	if queue.CurrentVideo.NextVideo == nil {
		queue.CurrentVideo = nil
	} else {
		queue.CurrentVideo = queue.CurrentVideo.NextVideo
	}
}

func (queue *VideoQueue) deleteSpecific(id string) (title string, err error) {
	// Loop through the linked list
	loopQueue := queue.CurrentVideo
	for loopQueue.NextVideo != nil {
		nextVideo := loopQueue.NextVideo
		// If next video's id equels to requested id
		if nextVideo.Id == id {
			// Cache the removed video title
			title = nextVideo.Title
			// Change the current video's next video to next next video
			loopQueue.NextVideo = nextVideo.NextVideo
			// Break the loop after deleted video
			break
		}
		loopQueue = nextVideo
	}
	// Error if the video is not in the queue
	if title == "" {
		println("Failed to delete video from the queue.")
		return "", errors.New("Video not exist.")
	}

	return title, nil
}

func (queue *VideoQueue) list() (listOfVideo string) {
	index := 1
	if queue.CurrentVideo == nil {
		return ""
	}

	// Init queue for loop through all video info
	loopQueue := queue.CurrentVideo

	// Next video until the end
	for loopQueue.NextVideo != nil || loopQueue.NextVideo == nil {
		listOfVideo += fmt.Sprintf("%d. %s\n", index, loopQueue.Title)
		// Break for loop if this is the last video
		if loopQueue.NextVideo == nil {
			break
		} else {
			loopQueue = loopQueue.NextVideo
			index += 1
		}
	}
	return listOfVideo
}

type Video struct {
	Id    string
	Title string
}

func (queue *VideoQueue) toArray() (arrayOfVideos []Video, err error) {
	if queue.CurrentVideo == nil {
		return nil, errors.New("Empty queue.")
	}

	// Loop through the linked list
	loopQueue := queue.CurrentVideo
	for loopQueue.NextVideo != nil || loopQueue.NextVideo == nil {
		// Append video array
		arrayOfVideos = append(arrayOfVideos, Video{
			Id:    loopQueue.Id,
			Title: loopQueue.Title,
		})
		// Break for loop if this is the last video
		if loopQueue.NextVideo == nil {
			break
		} else {
			loopQueue = loopQueue.NextVideo
		}
	}

	return arrayOfVideos, nil
}
