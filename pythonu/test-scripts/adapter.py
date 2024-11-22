import json
import base64
import os

import numpy as np


def numpy_encoder(obj):
    if isinstance(obj, np.ndarray):
        if obj.dtype == np.float32:
            data_type = 'float32'
            byte_data = obj.astype('<f4').tobytes()  # Convert to little-endian
        elif obj.dtype == np.float64:
            data_type = 'float64'
            byte_data = obj.astype('<f8').tobytes()  # Convert to little-endian
        else:
            raise TypeError(f"Unsupported dtype: {obj.dtype}")

        encoded_data = base64.b64encode(byte_data).decode('utf-8')
        result = {
            '_elementType': data_type,
            '_shape': obj.shape,
            '_data': encoded_data
        }
        return result
    raise TypeError(f'Object of type {obj.__class__.__name__} is not JSON serializable')


def numpy_decoder(dct):
    if '_elementType' in dct and '_data' in dct:
        data_type = dct['_elementType']
        shape = dct.get('_shape', None)
        data = base64.b64decode(dct['_data'])
        if data_type == 'float32':
            array = np.frombuffer(data, dtype='<f4')  # Convert from little-endian
        elif data_type == 'float64':
            array = np.frombuffer(data, dtype='<f8')  # Convert from little-endian
        else:
            raise TypeError(f"Unsupported element type: {data_type}")

        if shape:
            # print(array, flush =True)
            array = array.reshape(shape)
        return array

    return dct


def execute(funcs):
    rd, wd = 3, 4  # the read and write pipe indexes
    with os.fdopen(rd, "rb") as rf:
        os.write(wd, "ready".encode())
        while True:
            to_read = int.from_bytes(rf.read(4), "big")
            func_name_bytes = rf.read(to_read)
            func_name = func_name_bytes.decode()
            to_read = int.from_bytes(rf.read(4), "big")
            func_input = json.loads(rf.read(to_read), object_hook=numpy_decoder)

            # print(f'inputs: {func_input}, func name: {func_name}', flush=True)
            # call func
            result = funcs[func_name](func_input)

            msg_to_write = json.dumps(result, default=numpy_encoder).encode()
            x = int.to_bytes(len(msg_to_write), 4, "big")
            os.write(wd, x)
            os.write(wd, msg_to_write)