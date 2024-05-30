import map from 'it-map'
import { pipe } from 'it-pipe'
import { fromString as uint8ArrayFromString } from 'uint8arrays/from-string'
import { toString as uint8ArrayToString } from 'uint8arrays/to-string'
import { Transform } from 'stream'

export const protocol = "/Chat/1.0.0"

export function prt (val) {
  console.dir(val, {depth: null})
}

export function pipeMsg(connection, stream) {
  const peerId = connection.remotePeer.toString()
  const streamId = stream.id
  const msgBox = new Transform({
    transform(chunk, encoding, callback) {
      this.push(chunk);
      callback();
    }
  }) 
  
  pipe(
    () => {
      console.log("reveice from msgBox")
      return msgBox
    },
    (source) => map(source, (string) => {
      console.log(`reveice from msgBox.source: ${string}`)
      return uint8ArrayFromString(string)
    }),
    stream.sink
  )

  pipe(
    () => {
      console.log("reveice from stream")
      return stream.source
    },
    (source) => map(source, (buf) => {
      console.log(`reveice from stream.source: ${buf}`)
      return uint8ArrayToString(buf.subarray())
    }),
    async function (source) {
      for await (const msg of source) {
        console.log(`${peerId}.stream[${streamId}] > ` + msg.toString().replace('\n', ''))
      }
    }
  )

  const msgb = {
    id: peerId + "-" + streamId,
    chan: msgBox
  }

  return msgb
}