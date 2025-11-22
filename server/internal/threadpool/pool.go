package threadpool

import (
    "context"
    "fmt"
    "log"
    "sync"
    "time"
)

// Job represents a task to be executed
type Job struct {
    ID       string
    Task     func() error
    Priority int // Higher priority jobs are executed first
}

// WorkerPool manages a pool of workers
type WorkerPool struct {
    workers       int
    maxWorkers    int
    jobQueue      chan Job
    queueSize     int
    workerTimeout time.Duration
    activeWorkers int
    workersMu     sync.Mutex
    wg            sync.WaitGroup
    ctx           context.Context
    cancel        context.CancelFunc
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(workers, queueSize, maxWorkers int, workerTimeout time.Duration) *WorkerPool {
    ctx, cancel := context.WithCancel(context.Background())

    wp := &WorkerPool{
        workers:       workers,
        maxWorkers:    maxWorkers,
        jobQueue:      make(chan Job, queueSize),
        queueSize:     queueSize,
        workerTimeout: workerTimeout,
        activeWorkers: 0,
        ctx:           ctx,
        cancel:        cancel,
    }

    return wp
}

// Start starts the worker pool
func (wp *WorkerPool) Start() {
    log.Printf("Starting worker pool with %d workers (max: %d, queue: %d)", wp.workers, wp.maxWorkers, wp.queueSize)

    // Start initial workers
    for i := 0; i < wp.workers; i++ {
        wp.startWorker(i)
    }
}

// startWorker starts a single worker
func (wp *WorkerPool) startWorker(id int) {
    wp.workersMu.Lock()
    wp.activeWorkers++
    currentActive := wp.activeWorkers
    wp.workersMu.Unlock()

    wp.wg.Add(1)

    go func(workerID int) {
        defer wp.wg.Done()
        defer func() {
            wp.workersMu.Lock()
            wp.activeWorkers--
            wp.workersMu.Unlock()
        }()

        log.Printf("Worker %d started (total active: %d)", workerID, currentActive)

        idleTimer := time.NewTimer(wp.workerTimeout)
        defer idleTimer.Stop()

        for {
            select {
            case <-wp.ctx.Done():
                log.Printf("Worker %d shutting down", workerID)
                return

            case job := <-wp.jobQueue:
                // Reset idle timer when we get a job
                if !idleTimer.Stop() {
                    select {
                    case <-idleTimer.C:
                    default:
                    }
                }
                idleTimer.Reset(wp.workerTimeout)

                // Execute job
                if err := job.Task(); err != nil {
                    log.Printf("Worker %d: Job %s failed: %v", workerID, job.ID, err)
                } else {
                    log.Printf("Worker %d: Job %s completed", workerID, job.ID)
                }

            case <-idleTimer.C:
                // Worker has been idle for too long
                wp.workersMu.Lock()
                // Only kill worker if we have more than the minimum
                if wp.activeWorkers > wp.workers {
                    wp.workersMu.Unlock()
                    log.Printf("Worker %d idle timeout, shutting down", workerID)
                    return
                }
                wp.workersMu.Unlock()

                // Reset timer
                idleTimer.Reset(wp.workerTimeout)
            }
        }
    }(id)
}

// Submit submits a job to the worker pool
func (wp *WorkerPool) Submit(job Job) error {
    select {
    case <-wp.ctx.Done():
        return fmt.Errorf("worker pool is shutting down")
    default:
    }

    // Check queue size and spawn more workers if needed
    queueLen := len(wp.jobQueue)
    if queueLen > wp.queueSize/2 { // Queue is more than 50% full
        wp.workersMu.Lock()
        if wp.activeWorkers < wp.maxWorkers {
            newWorkerID := wp.activeWorkers
            wp.workersMu.Unlock()
            log.Printf("Queue %d%% full, spawning additional worker", (queueLen*100)/wp.queueSize)
            wp.startWorker(newWorkerID)
        } else {
            wp.workersMu.Unlock()
        }
    }

    // Submit job to queue
    select {
    case wp.jobQueue <- job:
        return nil
    case <-time.After(5 * time.Second):
        return fmt.Errorf("failed to submit job: queue full")
    }
}

// SubmitTask is a convenience method to submit a task without creating a Job
func (wp *WorkerPool) SubmitTask(id string, task func() error) error {
    return wp.Submit(Job{
        ID:       id,
        Task:     task,
        Priority: 0,
    })
}

// Shutdown gracefully shuts down the worker pool
func (wp *WorkerPool) Shutdown() {
    log.Println("Shutting down worker pool...")

    // Signal workers to stop
    wp.cancel()

    // Close job queue
    close(wp.jobQueue)

    // Wait for all workers to finish (with timeout)
    done := make(chan struct{})
    go func() {
        wp.wg.Wait()
        close(done)
    }()

    select {
    case <-done:
        log.Println("All workers shut down gracefully")
    case <-time.After(10 * time.Second):
        log.Println("Worker shutdown timeout reached")
    }
}

// GetStats returns worker pool statistics
func (wp *WorkerPool) GetStats() map[string]interface{} {
    wp.workersMu.Lock()
    defer wp.workersMu.Unlock()

    return map[string]interface{}{
        "active_workers": wp.activeWorkers,
        "min_workers":    wp.workers,
        "max_workers":    wp.maxWorkers,
        "queue_length":   len(wp.jobQueue),
        "queue_capacity": wp.queueSize,
        "queue_usage":    float64(len(wp.jobQueue)) / float64(wp.queueSize) * 100,
    }
}

// GetActiveWorkerCount returns the number of active workers
func (wp *WorkerPool) GetActiveWorkerCount() int {
    wp.workersMu.Lock()
    defer wp.workersMu.Unlock()
    return wp.activeWorkers
}

// GetQueueLength returns the current queue length
func (wp *WorkerPool) GetQueueLength() int {
    return len(wp.jobQueue)
}
