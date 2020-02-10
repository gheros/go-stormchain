# go-stormchain
Storm blockchain is developed by FileStorm. It is a multichain ecosystem that can support multiple blockchains including public, private and federated chains. It also supports multiple consensus.

## Storm公链

支持多种共识的多链架构区块链技术。信息公开，技术开源，可以为应用提供专属区块链，支持高TPS。自带区块链存储功能，使用方便。将于2020年第二季度发布。

## 风暴联盟链

FileStorm自主研发基于PBFT共识的区块链技术，支持国家监督和管控，多节点备份，数据存证不可篡改。以国密局标准算法加密原始数据，提取文件Hash上链传输，保护原文件及用户信息安全。

## 用户指南

下载运行程序 storm （选择正确的操作系统）
Linux: storm-linux(https://github.com/filestorm-fst/go-stormchain/blob/master/storm-linux)
Mac Os: storm-os(https://github.com/filestorm-fst/go-stormchain/blob/master/storm-os)

和创世文件 
storm_chain.json(https://github.com/filestorm-fst/go-stormchain/blob/master/storm_chain.json)

将 storm 和 storm_chain.json 复制到需要安装的节点服务器上。在这个指南中，我们将使用三台装有 Ubuntu 操作系统的服务器。（Storm区块链至少需要两个节点才能运行。）在服务器上，可以建立新文件夹 storm_node 作为节点运行根目录。将两个文件存到这个目录下。

### 第一步 建立账号

在区块链上，账号是用户身份的代表。每个节点程序都必须通过一个账号来运行。这个账号文件必须存在节点服务器上，密码由指定管理员保管。所以第一步，我们在三台服务器上各建一个账号。

``````````````````````````````````
$ cd storm_node
$ storm --datadir ./ account new
INFO [02-06|14:13:49.711] Maximum peer count                       FST=50 LES=0 total=50
Your new account is locked with a password. Please give a password. Do not forget this password.
Password:
Repeat password:

Your new key was generated

Public address of the key:   0x084149366635cD8E727d1eD50c363107b1b2a565
```````````````````````````````````
`0x084149366635cD8E727d1eD50c363107b1b2a565`是账号的公钥，可以公开。

使用 `--datadir ./` 的目的是把账号建立在节点运行根目录下，便于查找和管理。运行完后，将会看到一个新的名为 keystone 的文件夹。钥匙文件就在这个文件夹内。

在其他两个节点上也做同样的操作，并将密码保管好。三个公钥收集起来，等下需要使用。


### 第二步 生成密码文件

将密码写到一个密码文件中，方便节点启动。这个文件在节点启动后删除。

``````````````````````````````````
$ echo 'password' > password.txt
``````````````````````````````````

### 第三步 修改创世文件

```````````````````````
{
  "config": {
    "chainId": 20090103,
    "pbft": {
      "period": 7,
      "epoch": 36000
    }
  },
  "nonce": "0x0",
  "timestamp": "0x5de22b51",
  "extraData": "0x00000000000000000000000000000000000000000000000000000000000000001111111111111111111111111111111111111111222222222222222222222222222222222222222233333333333333333333333333333333333333330000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
  "gasLimit": "0x47b760",
  "difficulty": "0x1",
  "mixHash": "0x0000000000000000000000000000000000000000000000000000000000000000",
  "coinbase": "0x0000000000000000000000000000000000000000",
  "alloc":
  {
    "0000000000000000000000000000000000000000": {
      "balance": "0x200000000000000000000000000000000000000000000000000000000000000"
    }
  },
  "number": "0x0",
  "gasUsed": "0x0",
  "parentHash": "0x0000000000000000000000000000000000000000000000000000000000000000"
}
```````````````````````````````

这是缺省的创世文件。其中需要解释和修改的配置为：

chainId： Storm区块链是一个多链生态，每个区块链都有一个ID，20090103是主链的ID，（第一个区块链Bitcoin的诞生日为01/03/2009）每一条使用Storm区块链技术的新链都可以使用任何一个大于20090103，且未被使用过的整数做为chainId。所有的链都可以在 http://explorer.filestorm.info 上查到。

pbft：这是风暴联盟链使用的共识，未来Storm区块链将会支持更多的共识机制。

period：出块时间，缺省设定是7秒钟。可以根据实际情况修改。

extraData：创始区块验证人账号。因为Storm区块链启动的时候，至少需要两个验证人，所以要把创始区块验证人的账号记录在这里。注意：这只是创始区块验证人的账号设置。区块链跑起来以后，我们还可以通过让验证人投票的方式增加或减少验证人。我们可以用前面生成的三个账号去替代 extraData 值里的 1...111, 2...222, 3...333。在这里记得要把账号前的 0x 去掉。如果有更多创始区块验证人，可以把账号加到3...333后面。也可以减少，但是最少要有两个账号。

alloc：如果是有通证的区块链，可以在这里给一些特殊账号或者管理员账号一些通证。

修改完以后将文件保存好。记得在三个节点上，创世文件必须一模一样。

### 第四步 初始化节点

在三台节点服务器上运行下面的指令。

``````````````````````````````````
$ storm init storm_chain.json
``````````````````````````````````

### 第五步 启动节点

``````````````````````````````````
$ storm --datadir data  --networkid 20090103 --syncmode 'full' --port 30317 --unlock '0x084149366635cD8E727d1eD50c363107b1b2a565' --password password.txt --mine
``````````````````````````````````

--datadir data 是指定区块数据保存在当前目录下的 data 目录中。
--networkid 20090103 这个networkid必须和创世文件中的chainId相同。
--syncmode 'full' 是指同步方式为全同步。
--port 30317 这是 Storm 区块链的通讯端口。是一个可选项，缺省值为30303
--unlock '0x084149366635cD8E727d1eD50c363107b1b2a565' 这是解锁前面为这台服务器生成的管理员账号，让这个节点可以出块。
--password password.txt 使用存在密码文件中的密码。
--mine 启动挖矿（验证出块）

运行成功后，打开一个新窗口，将密码文件删除。

``````````````````````````````````
$ rm password.txt
``````````````````````````````````

这时候，我们会看到所有节点都停在第二个块
``````````````````````````````````
INFO [02-06|13:01:02.505] Commit new mining work                   number=1 sealhash=56a809…a06463 uncles=0 txs=0 gas=0 fees=0 elapsed=158.258µs
INFO [02-06|13:01:02.634] Successfully sealed new block            number=1 sealhash=56a809…a06463 hash=17d48d…cb74e4 elapsed=128.207ms
INFO [02-06|13:01:02.634] 🔨 mined potential block                  number=1 hash=17d48d…cb74e4
INFO [02-06|13:01:02.635] Commit new mining work                   number=2 sealhash=72548f…ba20b0 uncles=0 txs=0 gas=0 fees=0 elapsed=809.66µs
INFO [02-06|13:01:02.635] Signed recently, must wait for others
``````````````````````````````````
这是因为三个节点还没有互相连上，所以，下一步，就是要做节点的连接。


### 第六步 节点网络连接

选择一个节点做为连接主节点，在服务器上新开一个窗口，做如下操作，就可以进入节点控制面板。

``````````````````````````````````
$ cd storm_node
$ storm attach data/storm.ipc
``````````````````````````````````

然后在面板中做如下操作，就可以得到节点的网络ID
``````````````````````````````````
> admin.nodeInfo.enode
"enode://0e3a9317c4c9e8910e5d34f627ab798ac0814ac732cb433db4d73382a39c1cb2e8fd35fc53ad406f5883f86a45bc9fb083503ed357ed1a07b5c0a078815c534f@[::]:30317"
``````````````````````````````````

在另一个节点上打开节点控制面板
``````````````````````````````````
$ cd storm_node
$ storm attach data/storm.ipc
``````````````````````````````````
然后做如下操作，就可以将两个节点相连
``````````````````````````````````
> admin.addPeer("enode://0e3a9317c4c9e8910e5d34f627ab798ac0814ac732cb433db4d73382a39c1cb2e8fd35fc53ad406f5883f86a45bc9fb083503ed357ed1a07b5c0a078815c534f@[::]:30317")
``````````````````````````````````
记得在[::]填入正确的节点IP。如果是内网必须使用内网IP。

在第三个节点上也做同样操作，就可以将三个节点连起来。然后三个节点就开始轮流出块。


### 第七步 加新节点

启动节点
``````````````````````````````````
$ storm --datadir data  --networkid 20090103 --syncmode 'full' --port 30317 --unlock '0x084149366635cD8E727d1eD50c363107b1b2a565' --password password.txt --mine
``````````````````````````````````

在另一个窗口上打开节点控制面板
``````````````````````````````````
$ cd storm_node
$ storm attach data/storm.ipc
``````````````````````````````````
然后连接主节点，从而与整个网络相连
``````````````````````````````````
> admin.addPeer("enode://0e3a9317c4c9e8910e5d34f627ab798ac0814ac732cb433db4d73382a39c1cb2e8fd35fc53ad406f5883f86a45bc9fb083503ed357ed1a07b5c0a078815c534f@[::]:30317")
``````````````````````````````````

这时候，新节点还不能验证出块，只能接受区块信息。想要成为出块节点，就必须遵守所用共识机制。本指南采用的是pbft共识，增加新节点，必须通过验证节点投票，得到超过50%的赞同票就可以成为验证节点。下面，我们就来看看验证节点的投票功能。


### 第八步 验证节点投票

联盟链上的每一验证节点都可以代表一个独立的利益体。增加和减少验证节点都需要超过半数的验证节点投票决定。验证节点有一组投票功能，可以在控制面板中输入 pbft 来查看

``````````````````````````````````
> pbft
{
  votes: {},
  blockStatus: function(),
  dropVote: function(),
  getBlockStatusByHash: function(),
  getBlockStatusByNumber: function(),
  getValidatorsByHash: function(),
  getValidatorsByNumber: function(),
  vote: function()
}
``````````````````````````````````

  vote: 给一个新节点投赞成或者反对票。

  dropVote: 撤销给一个节点的投票。

  votes: 显示节点所有的投票。

  blockStatus: 显示区块状态，包括出块节点信息。

  getBlockStatusByHash: 通过区块哈希查找区块状态。

  getBlockStatusByNumber: 通过区块数查找区块状态。

  getValidatorsByHash: 通过区块哈希查找验证人信息。

  getValidatorsByNumber: 通过区块数查找验证人信息。

给一个节点投赞成票的指令是
``````````````````````````````````
> pbft.vote('0x53e5c08cb895599e7cfa5da58a783a56e9f140db', true)
``````````````````````````````````
投反对票是
``````````````````````````````````
> pbft.vote('0x53e5c08cb895599e7cfa5da58a783a56e9f140db', false)
``````````````````````````````````
投完票后可以撤销
``````````````````````````````````
> pbft.dropVote('0x53e5c08cb895599e7cfa5da58a783a56e9f140db')
``````````````````````````````````
如果在投票生效前撤销，相当于弃权，且不能再投票。如果投票已生效，撤销将没有意义。

查看在某个区块，如第300个块时的验证人列表，可以使用
``````````````````````````````````
> pbft.getValidatorsByNumber(300)
``````````````````````````````````
如果联盟成员达成共识，决定同时创建一条新链，可以通过第一到第六步生成节点创建新链。但是一旦链建成后，就必须用投票的方式增加新的验证节点。确保区块链的安全性。
