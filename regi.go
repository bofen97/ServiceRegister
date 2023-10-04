package ServiceRegister

import (
	"context"
	"log"
	"net"
	"sync"
	"time"

	"go.etcd.io/etcd/clientv3"
)

type RegisteClientService struct {
	Client        *clientv3.Client
	LeaseKeepChan <-chan *clientv3.LeaseKeepAliveResponse
	LeaseResp     *clientv3.LeaseGrantResponse
	Services      map[string]string //only client use it
	rlock         *sync.RWMutex
}

func (regs *RegisteClientService) ListenLaser() {

	for r := range regs.LeaseKeepChan {
		if r == nil {
			log.Print("laser closed .\n")
			return
		} else {
			log.Print("laser keeplive ok.\n")
		}
	}

}
func (regs *RegisteClientService) PutServiceAddr(key, value string) error {

	putresp, err := regs.Client.Put(context.Background(), key, value, clientv3.WithLease(regs.LeaseResp.ID))
	if err != nil {
		return err
	}
	log.Print(putresp.Header)
	return nil
}
func (regs *RegisteClientService) StoreKV(key, value string) {
	regs.rlock.Lock()
	defer regs.rlock.Unlock()
	log.Printf("store KV : %s  %s \n", key, value)
	regs.Services[key] = value

}

func (regs *RegisteClientService) GetK(key string) string {

	for {
		regs.rlock.RLock()
		value, ok := regs.Services[key]
		if ok {
			regs.rlock.RUnlock()
			return value
		}
		regs.rlock.RUnlock()
	}

}
func (regs *RegisteClientService) DelK(key string) {

	regs.rlock.Lock()
	log.Printf("[DELETE] should delete key %s\n", key)
	delete(regs.Services, key)
	regs.rlock.Unlock()

}

func (regs *RegisteClientService) WatchService(key string) {

	//
	getResp, err := regs.Client.Get(context.Background(), key)
	if err != nil {
		log.Fatal(err)
		return
	}
	if len(getResp.Kvs) > 0 {
		log.Printf("get server %s%s\n", getResp.Kvs[0].Key, getResp.Kvs[0].Value)
		regs.StoreKV(string(getResp.Kvs[0].Key), string(getResp.Kvs[0].Value))

	}

	wchan := regs.Client.Watch(context.Background(), key)

	for chaninfo := range wchan {
		for _, even := range chaninfo.Events {
			if even.Type.String() == "PUT" {
				regs.StoreKV(string(even.Kv.Key), string(even.Kv.Value))
			}
			if even.Type.String() == "DELETE" {
				regs.DelK(string(even.Kv.Key))
			}

		}
	}

}

func NewRegisteService(addr []string, timeNum int64) (*RegisteClientService, error) {

	client, err := clientv3.New(clientv3.Config{
		Endpoints:   addr,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		return nil, err
	}

	lease := clientv3.NewLease(client)
	leaseresp, err := lease.Grant(context.Background(), timeNum)
	if err != nil {
		return nil, err
	}
	leaseKeepChan, err := lease.KeepAlive(context.Background(), leaseresp.ID)

	if err != nil {
		return nil, err
	}

	return &RegisteClientService{Client: client, LeaseKeepChan: leaseKeepChan, LeaseResp: leaseresp, Services: make(map[string]string), rlock: &sync.RWMutex{}}, nil

}
func GetCurrentIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		log.Fatal(err)
	}
	var ipstr string
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				ipstr = ipnet.IP.String()
				break
			}
		}
	}
	return ipstr

}

// func main() {

// 	regs, err := NewRegisteService([]string{
// 		"localhost:2379",
// 	}, 5)
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	// go regs.ListenLaser()
// 	log.Printf("current ip %s\n", GetCurrentIP())
// 	regs.PutServiceAddr("query_topic", GetCurrentIP()+":8083")
// 	go regs.WatchService("redis")
// 	time.Sleep(1 * time.Second)

// 	go func() {
// 		for {
// 			time.Sleep(1 * time.Second)
// 			log.Printf("redis :%s\n", regs.GetK("redis"))
// 		}
// 	}()

// 	select {}

// }
