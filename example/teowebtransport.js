  // Delay function
  Promise.delay = time_ms => new Promise(resolve => setTimeout(resolve, time_ms));

  /** Service fields length in teowebtransport message. */
  const serviceFieldsLength = 4 + 4 + 1;

  /**
   * Encode a message into a Uint8Array, prefixing it with a 4-byte length.
   * 
   * @param {string} data
   * @return {Uint8Array}
   */
  function encodeMessage(id, command, data, err = 0) {

    // Encode command
    const commandEncoded = new TextEncoder().encode(command);
    const commandLength = commandEncoded.length;

    // Encode data
    const dataEncoded = new TextEncoder().encode(data);
    const dataLength = dataEncoded.length;

    // Calculate message length and create buffer
    const length = commandLength + dataLength + serviceFieldsLength;
    const buffer = new Uint8Array(4 + length);

    // Length of the message in 4 bytes
    buffer[0] = (length >>> 24) & 0xff;
    buffer[1] = (length >>> 16) & 0xff;
    buffer[2] = (length >>> 8) & 0xff;
    buffer[3] = length & 0xff;

    // ID of the message in 4 bytes
    buffer[4] = (id >>> 24) & 0xff;
    buffer[5] = (id >>> 16) & 0xff;
    buffer[6] = (id >>> 8) & 0xff;
    buffer[7] = id & 0xff;

    // Message data length in 4 bytes
    buffer[8] = (dataLength >>> 24) & 0xff;
    buffer[9] = (dataLength >>> 16) & 0xff;
    buffer[10] = (dataLength >>> 8) & 0xff;
    buffer[11] = dataLength & 0xff;

    // Message command
    buffer.set(commandEncoded, 12);

    // Message data
    if (dataLength > 0) {
      buffer.set(dataEncoded, 12 + commandLength);
    }

    // Message error
    buffer[12 + commandLength + dataLength] = err;

    return buffer;
  }

  /**
   * Get and decode entire message from buffer.
   * 
   * @param {Uint8Array} buffer
   * @return {null|{message: Uint8Array, newBuffer: Uint8Array}}
   */
  function getMessageFromBuffer(buffer) {
    // Not enough data in the buffer to read the 4-byte message length
    if (buffer.length < 4) {
      return null;
    }

    // Extract the 4-byte message length from the buffer
    const messageLength = buffer[0] << 24 | buffer[1] << 16 | buffer[2] << 8 | buffer[3];

    // Not enough data in the buffer to read the message
    if (buffer.length < 4 + messageLength) {
      return null;
    }

    // Extrract the message ID from the buffer
    const id = buffer[4] << 24 | buffer[5] << 16 | buffer[6] << 8 | buffer[7];

    // Extract the message data length from the buffer
    const dataLength = buffer[8] << 24 | buffer[9] << 16 | buffer[10] << 8 | buffer[11];

    const commandLength = messageLength - dataLength - serviceFieldsLength;

    // Extract the message command from the buffer
    const command = buffer.slice(12, 12 + commandLength);

    // Extract the message data from the buffer
    let data = new Uint8Array(0);
    if (dataLength > 0) {
      data = buffer.slice(12 + commandLength, 12 + commandLength + dataLength);
    }

    // Extract the message error from the buffer
    const err = buffer[12 + commandLength + dataLength];

    // Extract the remaining data from the buffer
    const newBuffer = buffer.slice(4 + messageLength);
    // const message = data;

    return { msg: { id, command, data, err }, newBuffer };
  }

  /**
   * Reads data from a ReadableStream and save data to buffer. If there is 
   * entire message in buffer than calls a callback function with the data.
   * 
   * @param {ReadableStream} stream
   * @param {string} errorText
   * @param {function(Uint8Array):void} dataReceivedFunction
   * @return {Promise<void>}
   */
  async function messageReader(stream, errorText, dataReceivedFunction) {
    try {
      let buffer = new Uint8Array(0);
      let reader = stream.getReader();
      while (true) {
        // Read data from stream
        data = await reader.read();
        if (data.done) {
          break;
        }

        // Add received data to the end of buffer
        const newBuffer = new Uint8Array(buffer.length + data.value.length);
        newBuffer.set(buffer);
        newBuffer.set(data.value, buffer.length);
        buffer = newBuffer;

        // Get messages from buffer
        next:
        while (true) {
          data = getMessageFromBuffer(buffer);
          if (data === null) {
            break next;
          }
          buffer = data.newBuffer;
          dataReceivedFunction(data.msg);
        }
      }
    } catch (error) {
      console.log(errorText + ':', error);
    }
  };

  /**
   * Reads data from a ReadableStream and calls a callback function with the data.
   * 
   * @param {ReadableStream} stream
   * @param {string} errorText
   * @param {function(Uint8Array):void} dataReceivedFunction
   * @return {Promise<void>}
   */
  async function streamReader(stream, errorText, dataReceivedFunction) {
    try {
      // Get a reader for the stream
      let reader = stream.getReader();

      // Read data from the stream until it is closed
      while (true) {
        // Read a chunk from the stream
        data = await reader.read();

        // If the chunk is the end of the stream, break out of the loop
        if (data.done) {
          break;
        }

        // Call the callback function with the chunk
        dataReceivedFunction(data.value);
      }
    } catch (error) {
      // Log any errors that occur
      console.log(errorText + ':', error);
    }
  };

  /**
   * Tries to connect to the WebTransport server. If the connection fails, it
   * will retry every 3 seconds.
   */
  const start = () => {
    connect().catch((err) => {
      console.error("can't connect:", err);
      setTimeout(start, 3000);
    })
  };

  async function connect(onconnect, ondisconnect, onmessage) {

    // Text encoder and decoder
    let encoder = new TextEncoder();
    let decoder = new TextDecoder();

    // Connect to WebTransport server
    console.log('Connecting to teowebtransport server...');
    let transport = new WebTransport("https://asuzs.teonet.dev:4433/wt");
    await transport.ready;

    // When the connection is closed
    transport.closed
      .then(() => console.log('Connection closed normally'))
      .catch(error => {
        console.log('Connection closed abruptly', error);
        start();
      });

    // Create client-initiated bidi stream & writer
    let stream = await transport.createBidirectionalStream();
    let writer = stream.writable.getWriter();

    // Create client-initiated uni stream & writer
    // let uniStream = await transport.createUnidirectionalStream();
    // let uniWriter = uniStream.getWriter();

    // Create datagram writer
    // let datagramWriter = transport.datagrams.writable.getWriter();

    // Display incoming datagrams
    // streamReader(transport.datagrams.readable, 'Datagram receive error', data => {
    //   console.log('Received datagram:', decoder.decode(data));
    // });

    // Display incoming data on bidi stream (messages mode)
    messageReader(stream.readable, 'Bidi stream receive error', msg => {
      if (msg.err) {
        console.log('Received on bidi stream, error:', decoder.decode(msg.data));
        return;
      }
      console.log('Received on bidi stream, data:', decoder.decode(msg.data));
    });

    // Display incoming bidi stream and data on those stream (when stream created by server)
    streamReader(transport.incomingBidirectionalStreams, 'Incoming bidi stream error', stream => {
      console.log('Received an incoming bidi stream');
      let incomingBidiWriter = stream.writable.getWriter();
      streamReader(stream.readable, 'Incoming bidi stream receive error', async data => {
        let text = decoder.decode(data);
        console.log('Received on incoming bidi stream:', text);
        await incomingBidiWriter.write(encoder.encode(text.toUpperCase()));
      });
    });

    // Display incoming uni stream and data on those stream (when stream created by server)
    // streamReader(transport.incomingUnidirectionalStreams, 'Incoming uni stream error', stream => {
    //   console.log('Received an incoming uni stream');
    //   streamReader(stream, 'Incoming uni stream receive error', data => {
    //     console.log('Received on incoming uni stream:', decoder.decode(data));
    //   })
    // });

    // Send some data on the streams we've created, wait, then send some more
    // await datagramWriter.write(encoder.encode("Datagram"))
    // await uniWriter.write(encoder.encode("Uni stream"))
    // console.log("Send message to bidi stream")
    // await writer.write(encodeMessage(0, "", "!!! Bidi stream AAA"))

    // await Promise.delay(1000);

    // await datagramWriter.write(encoder.encode("Datagram again"))
    // await uniWriter.write(encoder.encode("Uni stream again"))
    // await writer.write(encodeMessage(0, "", "Bidi stream again"))

    // await Promise.delay(1000);

    // Send some test commands
    for (let i = 0; i < 10; i++) {
      const msg = "привет message " + i;
      const name = Array(10)
        .fill('')
        .map((v, i, a) => String.fromCharCode(1040 + Math.floor(Math.random() * 33)))
        .join('')
        .toLowerCase();
      const upperCaseName = name[0].toUpperCase() + name.slice(1);
      console.log("Send command 'hello/" + upperCaseName + "' to bidi stream", msg)
      await writer.write(encodeMessage(i, "hello/" + upperCaseName, msg));
    }

    // await Promise.delay(2000);
    // await writer.close();

    // await Promise.delay(2000);
    // await transport.close();
  }

  start();
