# netlink源码分析
[Netlink 0007 --- 创建实现分析](https://blog.csdn.net/armlinuxww/article/details/99432380)

## netlink_proto_init源码分析
位置: RELATIVE:/net/netlink/af_netlink.c (Understand右键File最后Copy FullName)

#### 1. proto_register：
// 将全局netlink协议对象注册到 协议List中.  所以不能重复注册
挂载到/proc文件系统上。通过/proc/net/protocols可以看到注册协议的统计信息

// NETLINK协议netlink_proto是一个net包下面的全局对象

// nl_table也是一个全局对象


## sendto系统调用
sendto系统调用实际上还是找到该fd对应的socket实例，然后根据其socket类型调用ops->sendmsg;
    sock->ops->sendmsg(iocb, sock, msg, size);
