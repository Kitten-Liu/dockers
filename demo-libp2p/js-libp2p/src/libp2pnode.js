import process from 'node:process'
import { createLibp2p } from 'libp2p'
import { tcp } from '@libp2p/tcp'
import { noise } from '@chainsafe/libp2p-noise'
import { mplex } from '@libp2p/mplex'
import { yamux } from '@chainsafe/libp2p-yamux'
import { uPnPNAT } from '@libp2p/upnp-nat'
import { circuitRelayTransport, circuitRelayServer } from '@libp2p/circuit-relay-v2'
import { identify } from '@libp2p/identify'
import { autoNAT } from '@libp2p/autonat'
import { webSockets } from '@libp2p/websockets'
import * as filters from '@libp2p/websockets/filters'
import {pipeMsg, protocol, prt} from './streams.js'
import { dcutr } from '@libp2p/dcutr'

const node = await createLibp2p({
  addresses: {
    listen: ['/ip4/0.0.0.0/tcp/0/ws']
  },
  transports: [
    tcp(),  
    webSockets({
      filter: filters.all
    }),
    circuitRelayTransport({
      discoverRelays: 10
    })
  ],
  connectionEncryption: [noise()],
  streamMuxers: [
    yamux(),
    mplex()
  ],
  services: {
    identify: identify(),
    dcutr: dcutr(),
    // relay: circuitRelayServer({
    //   hopTimeout: 5 * 1000,
    //   reservations: {
    //     maxReservations: Infinity,
    //   }
    // }),
    // nat: uPnPNAT(),
  },
})

// start libp2p
await node.start()
console.log('libp2p has started')
console.log(`Node started with id ${node.peerId.toString()}`)

// print out listening addresses
console.log('listening on addresses:')
node.getMultiaddrs().forEach((addr) => {
  console.log(encodeURIComponent(addr.toString()))
})

// Wait for connection and relay to be bind for the example purpose
node.addEventListener('self:peer:update', (evt) => {
  // Updated self multiaddrs?
  node.getMultiaddrs().map((ma) => {
    const a = ma.toString()

    if (a.indexOf("p2p-circuit") !== -1) {
      console.log(`self:peer:update, Connected to the auto relay node via ${encodeURIComponent(a)}`);
    }
  })
})

node.msgBoxs=[]

await node.handle(protocol, async ({ connection, stream }) => {
  console.log(`new stream=${stream.id}`)
  const msgbox = pipeMsg(connection, stream)
  node.msgBoxs.push(msgbox)
})

export default node;