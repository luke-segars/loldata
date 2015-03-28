package queue

import (
	"fmt"
	gproto "github.com/golang/protobuf/proto"
	"github.com/kr/beanstalk"
	proto "proto"
	"time"
)

type QueueListener struct {
	Queue chan proto.ProcessedJobRequest

	conn    *beanstalk.Conn
	tubeset *beanstalk.TubeSet
}

/**
 * This function returns a channel that continuously pulls from the specified beanstalk queue
 * and returns jobs for a worker to work on.
 *
 * Closing the channel is safe and will stop listening to all tubes.
 * TODO: make closing the channel safe.
 */
func NewQueueListener(address string, tubes []string) (QueueListener, error) {
	listener := QueueListener{}
	conn, cerr := beanstalk.Dial("tcp", address)
	listener.Queue = make(chan proto.ProcessedJobRequest)

	if cerr != nil {
		return listener, cerr
	}

	// Create a new tube set and kick off a concurrent goroutine to continuously populate it.
	listener.tubeset = beanstalk.NewTubeSet(conn, tubes...)
	listener.conn = conn
	go harvestJobs(listener.tubeset, listener.Queue)

	return listener, nil
}

/**
 * Complete the current job and remove it from the queue.
 */
func (q *QueueListener) Finish(job proto.ProcessedJobRequest) {
	q.conn.Delete(*job.JobId)
}

func harvestJobs(ts *beanstalk.TubeSet, out chan proto.ProcessedJobRequest) {
	defer ts.Conn.Close()
	// ok := true

	for {
		// Jobs will be claimed by another worker if they exceed two hours runtime.
		id, body, err := ts.Reserve(2 * time.Hour)

		if err != nil {
			fmt.Println("Error: " + err.Error())
		//	close(out)
			// ok = false 
		} else {
			job := proto.ProcessedJobRequest{}

			gproto.Unmarshal(body, &job)
			job.JobId = gproto.Uint64(id)

			// Block until the current task is removed from the channel, then
			// pop another one on.
			out <- job
		}
	}
}
