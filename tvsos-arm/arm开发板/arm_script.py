import time
import sys
import requests
import threading
from pymodbus.client import ModbusSerialClient
from pymodbus.constants import Endian
from pymodbus.payload import BinaryPayloadDecoder

from requests.exceptions import ConnectionError, Timeout, RequestException
from flask import Flask, jsonify, request


# ----------------------- 配置后端系统信息 -----------------------------

BACKEND_HOST = '192.168.3.23'                       # 后端服务器地址
BACKEND_PORT = 8080                                 # 后端服务器端口
API_ENDPOINT = '/api/gps_data'                      # 后端API端点
HEADERS = {'Content-Type': 'application/json'}      # 请求头

# 串口配置
PORT = '/dev/ttyUSB0'                               # 串口设备路径 
BAUDRATE = 9600                                     # 波特率          
BYTESIZE = 8                                        # 数据位
PARITY = 'N'                                        # 校验位
STOPBITS = 1                                        # 停止位
TIMEOUT = 5                                         # 超时时间

# Modbus寄存器地址
LATITUDE_REG = 150                                  # 纬度寄存器起始地址
LONGITUDE_REG = 152                                 # 经度寄存器起始地址
SPEED_REG = 156                                     # 速度寄存器起始地址
NUM_REGISTERS = 2                                   # 读取的寄存器数量（每个浮点数占2个寄存器）

# -----------------全局Modbus客户端实例，供整个应用使用--------------------

modbus_client = None

#------------------Modbus客户端连接管理-------------------------------

def read_gps_data_modbus() -> tuple[float | None, float | None, float | None]:
    
    global modbus_client        # 使用全局的Modbus客户端实例

    if not modbus_client or not modbus_client.connected:
        print("Modbus客户端未连接或已断开，尝试重新连接...")
        try:
            modbus_client.close() # 尝试关闭可能存在的旧连接
            if modbus_client.connect():
                print("Modbus客户端重新连接成功。")
            else:
                print("Modbus客户端重新连接失败！")
                return None, None
        except Exception as e:
            print(f"Modbus重新连接时发生错误: {e}")
            return None, None

    latitude = None         #初始化经纬度为None
    longitude = None

    try:
        # 读取纬度寄存器（寄存器地址，数量，slave默认为1）
        result_lat_registers = modbus_client.read_holding_registers(address=LATITUDE_REG, count=NUM_REGISTERS, slave=1)
        if result_lat_registers.isError():
            print(f"读取纬度寄存器失败: {result_lat_registers}")
            return None, None 

        # 解析寄存器数据为浮点数
        decoder_lat = BinaryPayloadDecoder.fromRegisters(
            result_lat_registers.registers,
            byteorder=Endian.BIG,
            wordorder=Endian.BIG
        )
        latitude = decoder_lat.decode_32bit_float()     #解析为32位浮点数

        # 读取经度寄存器（寄存器地址，数量，slave默认为1）
        result_lon_registers = modbus_client.read_holding_registers(address=LONGITUDE_REG, count=NUM_REGISTERS, slave=1)
        if result_lon_registers.isError():
            print(f"读取经度寄存器失败: {result_lon_registers}")
            return None, None 
        # 解析寄存器数据为浮点数
        decoder_lon = BinaryPayloadDecoder.fromRegisters(
            result_lon_registers.registers,
            byteorder=Endian.BIG,
            wordorder=Endian.BIG
        )
        longitude = decoder_lon.decode_32bit_float()    #解析为32位浮点数

        # 读取速度寄存器（寄存器地址，数量，slave默认为1）
        result_spd_registers = modbus_client.read_holding_registers(address=SPEED_REG, count=NUM_REGISTERS, slave=1)
        if result_spd_registers.isError():
            print(f"读取速度寄存器失败: {result_spd_registers}")
            return None, None
        # 解析寄存器数据为浮点数
        decoder_spd = BinaryPayloadDecoder.fromRegisters(
            result_spd_registers.registers,
            byteorder=Endian.BIG,
            wordorder=Endian.BIG
        )
        speed = decoder_spd.decode_32bit_float()    #解析为32位浮点数


        if latitude is not None and longitude is not None and speed is not None:
            print(f"Module GPS: Latitude = {latitude:.6f}, Longitude = {longitude:.6f}, Speed = {speed:.2f}")
        return latitude, longitude, speed*1.852  # 转换为km/h

    except Exception as e:
        print(f"读取或解析GPS数据出错: {e}")
        return None, None, None



#---------------------------------HTTP连接测试----------------------------------

def test_http_connection(timeout: int = 5) -> tuple[bool, str]:

    url = f"http://{BACKEND_HOST}:{BACKEND_PORT}"   # 后端 URL
    print(f"测试与后端的HTTP连接: {url}")
    
    try:
        response = requests.get(url, timeout=timeout, verify=False) # 检测连接
        return True, f"HTTP连接成功。URL: {url}, 状态码: {response.status_code}"
    except ConnectionError:
        return False, f"连接错误: 无法连接到 {BACKEND_HOST}:{BACKEND_PORT}。请检查地址、端口或网络连通性。"
    except Timeout:
        return False, f"连接超时: 在 {timeout} 秒内未能连接到 {BACKEND_HOST}:{BACKEND_PORT}。"
    except RequestException as e:
        return False, f"请求异常: 连接到 {BACKEND_HOST}:{BACKEND_PORT} 时发生未知错误: {e}"
    except Exception as e:
        return False, f"未知错误: {e}"


#--------------------------------向后端发送数据--------------------------------------

def send_to_backend(data):
  
    url = f"http://{BACKEND_HOST}:{BACKEND_PORT}{API_ENDPOINT}"     # 完整的后端URL
    print(f"尝试向 {url} 发送数据...")

    try:
        response = requests.post(url, json=data, headers=HEADERS, timeout=5)    # 发送POST请求
        response.raise_for_status() # 如果响应状态码不是200-299，将引发HTTPError
        print(f"数据发送成功！Backend响应: {response.status_code} {response.text}")
        
    except requests.exceptions.ConnectionError as e:
        print(f"发送数据失败，无法连接到后端 {url}: {e}")
    except requests.exceptions.Timeout as e:
        print(f"发送数据失败，请求超时到后端 {url}: {e}")
    except requests.exceptions.HTTPError as e:
        print(f"发送数据失败，后端返回HTTP错误 {e.response.status_code}: {e.response.text}")
    except requests.exceptions.RequestException as e:
        print(f"发送数据时发生未知错误: {e}")
    except Exception as e:
        print(f"发送数据时发生异常: {e}")

#-----------------------------Flask应用及路由--------------------------------------

app = Flask(__name__)

#-----------------------------处理GPS请求的线程-------------------------------------

def process_gps_request_in_thread():

    lat, lon, spd = read_gps_data_modbus() # 获取GPS数据

    if lat is not None and lon is not None:
        # 构建json数据
        payload = {
            "latitude": round(lat, 6),
            "longitude": round(lon, 6),
            "speed": round(spd, 2),
            "device_id": "MyGPS_Device",
            "timestamp": time.time()
        }
        send_to_backend(payload)
    else:
        print("未获取到有效的GPS数据，跳过发送到后端。")
        
#-----------------------------Flask路由定义--------------------------------------

@app.route('/notify_request_received', methods=['POST', 'GET']) # 后端请求GPS数据的路由

def listen_and_send():

    print("收到后端请求，将在后台线程中处理GPS数据...")
    
    # 启动一个新线程来处理GPS数据读取和发送
    thread = threading.Thread(target=process_gps_request_in_thread)
    thread.start()

    return jsonify({"status": "acknowledged", "message": "请求已接收，GPS数据正在后台处理并发送中。"}), 200

if __name__ == "__main__":

    # 初始化全局Modbus客户端
    modbus_client = ModbusSerialClient(
        port=PORT,
        baudrate=BAUDRATE,
        bytesize=BYTESIZE,
        parity=PARITY,
        stopbits=STOPBITS,
        timeout=TIMEOUT
    )

    if not modbus_client.connect():
        print(f"错误：无法连接到串口 {PORT}。请检查设备是否连接或权限。")
        sys.exit(1)
    print(f"成功连接到Modbus RTU设备在 {PORT}")

    while True:
        if_http_isconnected, http_msg = test_http_connection()
        if not if_http_isconnected:
            print(f"与后端的http连接失败，请检查网络设置或后端服务状态，\n错误信息：{http_msg}")
            print("准备进行重试...")
            time.sleep(5)
        else:
            print("与后端连接成功，本地服务将监听后端位置获取请求。")
            break

    app.run(host='192.168.3.10', port=5000, debug=False)  # 启动Flask应用，监听所有可用接口的5000端口

    if modbus_client and modbus_client.connected: # 关闭Modbus连接
        modbus_client.close()
        print("Modbus客户端已关闭。")