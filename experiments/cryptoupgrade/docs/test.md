# R15占用问题-协处理器优化方案

 **System V AMD64 ABI**（x86-64 架构下的 C 语言调用约定）中，R15 寄存器被定义为 **被调用者保存寄存器（callee-saved register）**。这意味着：

- 如果一个 C 函数使用了R15 寄存器，它必须在函数返回之前恢复 R15的原始值。
- 调用 C 函数的代码可以假设 R15 的值在函数调用前后保持不变。

但是在 Go 的运行时中，R15 寄存器通常被用作 **goroutine 的上下文指针**，指向当前 goroutine 的结构体。

## 更换方案

- Go Lua：编译型语言编译后功能确定，而解释性语言实现动态更新更为简单，利用go语言调用解释性语言可以实现这类效果
- Go plugin + rpc:基于RPC的插件机制
- CGO+C DLL:利用c语言DDL库实现，利用go语言触发。
- go原生提供的方法： go install -buildmode=shared -linkshared  std

| 问题 | 微服务 | Go plugin     |  RPC plugin   | CGO(C语言RPC) |
| -- | ---- | ---- | ---- |----|
| 优点 | go语言原生效率更高 | 完全动态编译，不需要与其他进程或者操作系统进行交互 | 基于RPC实现的plugin，需要与其他进程交互，近似实现了动态更新的效果 |  |
| 缺点   | go语言 | 1. R15，CX兼容性问题，核心问题是GO的buildmode限制了实现 | 基于RPC的实现，可能无法抵抗侧信道攻击 ，虚拟机逃逸 |1. 类似Go plugin，依然可能会存在各种兼容性问题IPC|
| 解决方案 |      | Go 编译器层面的修改                                     |  |目前看来比较可行的方案|

go编译器层面深入

WASM

执行证明、防篡改（内容）证明

## 虚拟机逃逸

虚拟机逃逸指的是攻击者利用虚拟机软件或宿主机操作系统的漏洞，突破虚拟机的隔离环境，从虚拟机内部访问或控制宿主机系统，或者进一步影响同一宿主机上的其他虚拟机的行为。

测信道攻击属于常见的虚拟机逃逸策略，利用硬件层面差异，针对密码算法进行分析。

## 动态库编译模式与Plugin

**`shared` 编译模式**：更侧重于将包编译为动态库，然后在编译主程序时进行链接，适用于需要在多个程序之间共享代码库的场景。

**`go plugin` 机制**：强调在程序运行时动态加载插件，使得程序可以在不重新编译的情况下扩展功能，适用于需要灵活扩展功能的场景，如插件系统。

## 资料：

[bug: When dynamic linking, R15 may be clobbered by a global variable access · Issue #468 · Consensys/gnark-crypto](https://github.com/Consensys/gnark-crypto/issues/468)  将需要编译的内容作为远程库，能够解决冲突，但是不能够解决停机

# 未完成功能

1. 一个算法由多个文件组成（密码库构建）
2. Plugin严格要求编译环境相同（Geth编译时的参数）
3. 复杂数据类型的序列化方法
4. 动态编译重新启动问题
6. 汇编语言支持问题

# Go plugin学习与测试

plugin编译命令：

```shell
# 将指定的包编译为plugin
go build -buildmode=plugin -tags=purego -o <outputPath>  <packagename>
# 编译当前目录下main包作为plugin
go build -buildmode=plugin -tags=purego -o <outputPath> .
```

可携带参数：

```shell
-ldflags "-X github.com/ethereum/go-ethereum/internal/version.gitCommit=22064197a863ef3eb94829f3a5a5704e848eb30a -X github.com/ethereum/go-ethereum/internal/version.gitDate=20241204 -extldflags '-Wl,-z,stack-size=0x800000'" 
-tags urfave_cli_no_docs,ckzg 
-trimpath -v -o /home/ubuntu/project/modifyGeth/build/bin/geth ./cmd/geth
```

可能引发Plugin与主程序不一致的选项：

1. trimpath

不会引发不一致的选型：

1. ldflags其中：`-ldflags` 是传递给 Go 链接器(go linker)的标志`-extldflags` 是 Go 链接器再传递给外部链接器(通常是 gcc/ld)的标志.它们是通过嵌套方式工作的，但不是简单的包含关系.后者必须在前者的字符串中存在。

# Add算法测试

## 算法准备

```go
package main

func Add(a int, b int) int {
	return a + b
}
```

代码文件必须使用package main,而后将整个文件通过gzip压缩并转化为base64编码：

H4sIAAAAAAAA/ypITM5OTE9VyE3MzOPiyswtyC8qUdDg4lTKTSzJ0E/KTFfi0uTiSivNS1ZwTEnRSFTQSspM1/PMK9FRSIKzNeEshWouzqLUktKiPIVEPbAGHYUkTa5aLkAAAAD//9dFMqpoAAAA

codestorage contract ABI通过  [abi压缩网站](https://www.bejson.com/zhuanyi/) 实现代码压缩到一行

```json
[{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"string\",\"name\":\"name\",\"type\":\"string\"}],\"name\":\"codeUploaded\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"name\",\"type\":\"string\"},{\"internalType\":\"bytes\",\"name\":\"input\",\"type\":\"bytes\"}],\"name\":\"callFunc\",\"outputs\":[{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"name\",\"type\":\"string\"}],\"name\":\"getCode\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"name\",\"type\":\"string\"}],\"name\":\"getGas\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"name\",\"type\":\"string\"}],\"name\":\"getInfo\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"code\",\"type\":\"string\"},{\"internalType\":\"uint64\",\"name\":\"gas\",\"type\":\"uint64\"},{\"internalType\":\"string\",\"name\":\"itype\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"otype\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"name\",\"type\":\"string\"},{\"internalType\":\"uint64\",\"name\":\"_gas\",\"type\":\"uint64\"}],\"name\":\"updataGas\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"name\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"code\",\"type\":\"string\"},{\"internalType\":\"uint64\",\"name\":\"gas\",\"type\":\"uint64\"},{\"internalType\":\"string\",\"name\":\"itype\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"otype\",\"type\":\"string\"}],\"name\":\"uploadCode\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]
```

## 直接调用

注意算法名字要大写

```shell
# 启动，注意账户地址是否合法
./geth --datadir chain/node1 --port 30666  --networkid 1 --unlock 57F96028bA3258ebFb4940d67443967cF23e3fc4 --password chain/password.txt --authrpc.port 8556 --allow-insecure-unlock

#部署合约阶段
abi=JSON.parse("填入压缩转义后的abi")
CodeStorage=eth.contract(abi)
codestorage=CodeStorage.at("0000000000000000000000000000000000000043")
# 上传代码
codestorage.uploadCode("Add","H4sIAAAAAAAA/ypITM5OTE9VyE3MzOPiyswtyC8qUdDg4lTKTSzJ0E/KTFfi0uTiSivNS1ZwTEnRSFTQSspM1/PMK9FRSIKzNeEshWouzqLUktKiPIVEPbAGHYUkTa5aLkAAAAD//9dFMqpoAAAA",1,"int256,int256","int256",{from:eth.accounts[0]})
#调用
codestorage.callFunc("Add","0x00000000000000000000000000000000000000000000000000000000000000640000000000000000000000000000000000000000000000000000000000000064",{from:eth.accounts[0]})
```
```shell
codestorage.uploadCode("Sum256","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",1,"bytes","bytes32",{from:eth.accounts[0]})

codestorage.callFunc("Sum256","0x0000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000c48656c6c6f20776f726c64210000000000000000000000000000000000000000",{from:eth.accounts[0]})
```

## 合约间调用

调用合约编写

```solidity
contract interCall{
    address codeStoraddress=0x0000000000000000000000000000000000000043;
    string codeSorSign="callFunc(string)";    
    function verifyBallotzkp(string memory p1,string memory p2) external returns(int256){
        string memory funName="verifyBallotzkp";
        // abi.encode序列化参数，参数可以添加多个，
        bytes memory input=abi.encode(p1,p2);
        (bool success,bytes memory output)=codeStoraddress.call(abi.encodeWithSignature(codeSorSign,funName,input));
        require(success,"Low-level call error");
        //  返回值反序列化为bool类型
        bool result=abi.decode(output,(bool));
        require(result,"Verify failed")
        return c;
    }
    
    function verifyBallotzkp(string memory data) external returns(int256){
        string memory funName="verifyBallotzkp";

        (bool success,bytes memory output)=codeStoraddress.call(abi.encodeWithSignature(codeSorSign,funName,data));
        require(success,"Low-level call error");
        //  返回值反序列化为bool类型
        bool result=output;
        require(result,"Verify failed")
        return c;
    }
    
 
	    function test(int256 a, int256 b) external returns(int256){
        string memory funName="add";
        // 手动构造参数
        bytes memory input=abi.encode(a,b);
        // 调用
        (bool success,bytes memory output)=codeStoraddress.call(abi.encodeWithSignature(codeSorSign,funName,input));
        require(success,"Low-level call error");
        // 手动处理返回值
        int256 c=abi.decode(output,(int256));
        return c;
    }
}

// Solidity 对bytes类型的赋值
bytes memory data = hex"68656c6c6f"; 
```

abi:

[{\"inputs\":[{\"internalType\":\"int256\",\"name\":\"a\",\"type\":\"int256\"},{\"internalType\":\"int256\",\"name\":\"b\",\"type\":\"int256\"}],\"name\":\"test\",\"outputs\":[{\"internalType\":\"int256\",\"name\":\"\",\"type\":\"int256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]

测试：

```cmd
#正常发起交易
intercall.test(200,100,{from:eth.accounts[0]})
# 通过call获取返回值
intercall.test.call(200,100,{from:eth.accounts[0]})
```

# 密码算法测试

## Hash

1. blake2b ，测试函数：Sum256
2. blake2s，

## Sign

## Encode

## Commit ： curiska，kzg

| ALgorithm  | Success | Note                            |
| :--------: | ------- | ------------------------------- |
|   bls.go   | no      | assembly language unsupportable |
| blake2b.go | yes     |                                 |
|            |         |                                 |
|            |         |                                 |

# 链外执行器

合约间调用，拉起第一个交易。

参数类型：

string,int256,string,int32(12)

# 治理验证函数	

利用preload实现预装载。

# BUG解决

## 支持的数据类型与对应关系

| SOLIDITY | GO       |
| :------: | -------- |
|  int64   | int64    |
|  int256  | *big.int |
| bytes32  | [32]byte |
|  bytes   | []byte   |

不支持的类型：

1. int：go的int是跨平台支持的，因此其长度是不固定的，这对于solidity来说会产生错误，建议以int256方式使用。
1. bytes33及以上的chang'du

## Plugin不兼容问题

考虑是-tags问题

```shell
# geth 编译命令
/usr/local/go/bin/go build -ldflags "-X github.com/ethereum/go-ethereum/internal/version.gitCommit=22064197a863ef3eb94829f3a5a5704e848eb30a -X github.com/ethereum/go-ethereum/internal/version.gitDate=20241204 -extldflags '-Wl,-z,stack-size=0x800000'" -tags urfave_cli_no_docs,ckzg -trimpath -v -o /home/ubuntu/project/modifyGeth/build/bin/geth ./cmd/geth

# 简化命令
/usr/local/go/bin/go build -v -o /home/ubuntu/project/modifyGeth/build/bin/geth ./cmd/geth

# plugin编译命令
go build buildmodule=
```

## Plugin占用R15寄存器问题

解决方案：cgo调用

# 统一序列化方法

传输结构非常复杂，结构体嵌套多层。

多虚拟机管理。

解决方案：

1. 内联汇编，solidity直接调用EVM指令
2. EVM修改，gas降低，持久化存储
3. 直接载入EVM内存，读取困难？
