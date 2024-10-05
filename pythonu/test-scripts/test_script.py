import numpy as np

from pythonu.adapter import execute


def add(i):
    a = i['a']
    b = i['b']
    return {'result': a + b}


def add_scalar_output(i):
    a = i['a']
    b = i['b']
    return a+b


def add_numpy_arrays(i):
    a = i['a']
    b = i['b']
    return a+b


def identity(i):
    return i


def verify_2d_array(i):
    arr = i['arr2D']
    expected = np.array([[1.2,3.2],[99.1,-14.1]])
    if not np.array_equiv(arr, expected):
        raise Exception(f"expected arr {expected} but was {arr}")
    return i


def verify_1d_array(i):
    arr = i['arr1D']
    expected = np.array([1.2,3.2, 99.1,-14.1])
    if not np.array_equiv(arr, expected):
        raise Exception(f"expected arr {expected} but was {arr}")
    return i


if __name__ == '__main__':
    execute(globals())