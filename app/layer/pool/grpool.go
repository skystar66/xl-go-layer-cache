package pool

import (
	"g2cache/app/layer/helper"
	"github.com/gogf/gf/os/glog"
	"runtime"
	"sync"
	"time"
)

//表示一个用户请求任务
type job func()

//表示线程池
type worker struct {
	id   int
	pool *Pool
}
type Pool struct {

	//任务队列，类型是 job,go中的任务队列通过 channel表示
	jobQueue chan job
	//线程数量
	workers  []*worker
	stopOnce sync.Once
	stopped  chan struct{}
	//线程阻塞等待
	wg sync.WaitGroup
}

var (
	onceinstance *Pool
	once         sync.Once
)

//创建线程池, workNumbers:线程数量,jopQueueLength:任务队列大小限制
func NewPool(workNumbers int, jopQueueLength int) *Pool {
	once.Do(func() {
		pool := Pool{
			jobQueue: make(chan job, jopQueueLength),
			workers:  make([]*worker, workNumbers),
			stopped:  make(chan struct{}, 1),
		}

		for i := 0; i < workNumbers; i++ {
			//创建线程
			pool.workers[i] = NewWorker(i, &pool)
			//添加线程
			pool.wg.Add(1)
		}

		pool.wg.Add(1)
		//添加监控
		go pool.monitor()

		onceinstance = &pool
	})
	return onceinstance
}

//创建线程
func NewWorker(id int, pool *Pool) *worker {

	worker := worker{
		id:   id,
		pool: pool,
	}
	worker.start()
	return &worker

}

//线程执行
func (w *worker) start() {
	go func() {
		glog.Infof("Pool [%d] worker start run ...", w.id)
		defer w.pool.wg.Done()
		//监听队列消息
		w.listener()
	}()
}

//监听队列消息，进行消费
func (w *worker) listener() {
	for {
		select {
		case <-w.pool.stopped:
			glog.Infof("Pool [%d] worker stop run ...", w.id)
			//将剩余任务全部执行
			if len(w.pool.jobQueue) > 0 {

				for job := range w.pool.jobQueue {
					//执行任务
					runJob(w.id, job)
				}
			}
			return
		case job, ok := <-w.pool.jobQueue:
			if ok {
				runJob(w.id, job)
			}
		}
	}

}

//执行任务
func runJob(id int, f func()) {
	defer func() {
		//使用recover 配合defer 捕获panic异常
		if err := recover(); err != nil {
			glog.Infof("Pool [%d] Job panic err: %v, stack: %v\n", id, err, string(outputStackErr()))
		}
	}()
	//执行业务
	f()
}

//封装job,//todo 其实没什么卵用，只是提供一个口子，后期如果想加点什么东西 再说吧
func (p *Pool) wrapperJob(job func()) func() {
	return job
}

//发送job任务，带有超时时间的
func (p *Pool) SendJobWithTimeout(job func(), timeout time.Duration) bool {

	select {
	case <-p.stopped:
		return false

	case <-time.After(timeout):
		return false
	default:

	}
	p.jobQueue <- job
	return true
}

//发送延迟job任务
func (p *Pool) SendJobWithDelay(job func(), timeout time.Duration) bool {
	select {
	case <-p.stopped:
		return false
	case <-time.After(timeout):
		p.jobQueue <- job
		return true
	default:

	}
	return false
}

//todo 后期考虑实现延迟队列
func (p *Pool) SendJob(job func()) bool {
	select {
	case <-p.stopped:
		return false
	default:

	}
	p.jobQueue <- job
	return true
}

func outputStackErr() []byte {
	var (
		buf [4096]byte
	)
	n := runtime.Stack(buf[:], false)
	return buf[:n]
}

//添加监控
func (p *Pool) monitor() {
	t := time.NewTicker(time.Duration(helper.CacheMonitorJobQueueSecond) * time.Second)
	for {
		select {
		case <-p.stopped:
			t.Stop()
			return
		case <-t.C:
			glog.Infof("Pool jobQueue current len %d", len(p.jobQueue))
		}
	}
}

//todo 释放
func (p *Pool) release() {
	close(p.stopped)
	force := make(chan struct{})
	forceOne := sync.Once{}
	go func() {
		for {
			select {
			case <-force:
				return
			default:
				p.wg.Wait() // why always some goroutine not exit,who found bug
				forceOne.Do(func() {
					close(force)
				})
				return
			}
		}
	}()
	// forceExit
	time.AfterFunc(5*time.Second, func() {
		forceOne.Do(func() {
			close(force)
		})
	})
	<-force
	close(p.jobQueue)
}
