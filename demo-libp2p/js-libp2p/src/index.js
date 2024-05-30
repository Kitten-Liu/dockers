import express from 'express';
import libp2pNode from './libp2pnode.js';
import { program } from 'commander';
import { multiaddr } from '@multiformats/multiaddr'
import { fromString, toString } from 'uint8arrays'
import {pipeMsg, prt, protocol} from './streams.js'
import { Stream } from 'stream';

program.option("-p --port <int>", "http port", 3000);
program.parse();

const options = program.opts()
const app = express();
const port = options.port || 3000;

app.listen(port, () => {
  console.log(`Server running on port ${port}`)
});

app.get('/relay/:multiaddr', async (req, res) => {
    try {
      const addr = req.params.multiaddr || '';
      const peer = multiaddr(addr);
      console.log(`relay peer=${peer}`)
      
      await libp2pNode.dial(peer).then((conn) => {
        res.send(`relay.dial.status=${conn.status}`);
      }).catch((reason) => {
        res.send(`relay.dial.fail=${reason}`);
      });
    } catch (error) {
      res.status(500).send(`Failed to connect: ${error.message}`);
    }
});

app.get('/connect/:multiaddr', async (req, res) => {
    try {
      const addr = req.params.multiaddr || '';
      const peer = multiaddr(addr);
      console.log(`connect peer=${peer}`)

      var dialOpt = {
        runOnTransientConnection: false,
        negotiateFully: true,
      }

      if (addr.indexOf("/p2p-circuit/") !== -1) {
        dialOpt.runOnTransientConnection = true
        dialOpt.negotiateFully = false
      }
      
      const conn = await libp2pNode.dial(peer)
      const stream = await conn.newStream(protocol, dialOpt)
      const msgbox = pipeMsg(conn, stream)
      libp2pNode.msgBoxs.push(msgbox)

      res.send(`Connected to peer at ${addr}`);
    } catch (error) {
      res.status(500).send(`Failed to connect: ${error.message}`);
    }
});

app.get('/addrs', async (req, res) => {
  const newAddrs = libp2pNode.getMultiaddrs()
  newAddrs.map((ma) => {
    console.log(`Connected to the auto relay node via ${ma.toString()}`);
  })
  res.send(`Connected to the auto relay node via ${newAddrs}`);
});

app.get('/message/:msg', async (req, res) => {
    const { msg } = req.params;
    const b = fromString(msg + "\n")

    libp2pNode.msgBoxs.map(({id, chan, stream}) => {
      chan.write(msg + "\n", "utf8", (err) => {
        if (err) {
          console.log(`err=${err}`)
        }
      })
    })
    res.send(`Message sent: ${msg}`)
});