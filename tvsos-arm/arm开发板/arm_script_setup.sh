#!/bin/bash

echo "开始设置环境并运行"

# 1. 更新apt源
echo "更新apt包列表..."
sudo apt update
if [ $? -ne 0 ]; then
    echo "错误：apt更新失败。请检查网络连接或apt配置。"
    exit 1
fi
echo "apt包列表更新完成。"

# 2. 安装Python3和pip3
echo "安装Python3和pip3..."
sudo apt install python3 -y
if [ $? -ne 0 ]; then
    echo "错误：Python3安装失败。"
    exit 1
fi
echo "Python3和pip3安装完成。"

# 3. 安装Python依赖
echo "安装Python依赖 (pymodbus, requests, flask, waitress)..."
sudo apt install python3-pymodbus python3-requests python3-flask python3-waitress -y
if [ $? -ne 0 ]; then
    echo "错误：Python依赖安装失败。请检查系统包管理器"
    exit 1
fi
echo "Python依赖安装完成。"

# 4. 添加用户到dialout组以访问串口
echo "设置串口访问权限..."
# 检查当前用户是否已在dialout组
if grep -q "dialout" /etc/group | grep -q "$USER"; then
    echo "当前用户 '$USER' 已在 'dialout' 组中，无需额外设置。"
else
    echo "将当前用户 '$USER' 添加到 'dialout' 组以获得串口访问权限。"
    sudo usermod -aG dialout $USER
    echo "权限更改需要重新登录才能生效。如果您是第一次运行此脚本，请在脚本完成后重新登录系统（或执行 'newgrp dialout'）。"
fi
echo "串口访问权限设置完成。"

# 5. 保存Python脚本
PYTHON_SCRIPT_NAME="arm_script.py"
echo "创建Python脚本文件 '$PYTHON_SCRIPT_NAME'..."
cat << 'EOF' > "$PYTHON_SCRIPT_NAME"
import time  
import datetime
import sys  
import os  
import requests  
import threading  
import logging  
from pymodbus.client import ModbusSerialClient  
from pymodbus.constants import Endian  
from pymodbus.payload import BinaryPayloadDecoder  
from requests.exceptions import ConnectionError, Timeout, RequestException  
from flask import Flask, jsonify, request  
from waitress import serve # 新增导入
  
# ----------------------- 配置后端系统信息 -----------------------------  
BACKEND_HOST = '192.168.3.23'                       # 后端服务器地址  
BACKEND_PORT = 8080                                 # 后端服务器端口  
API_ENDPOINT = '/api/gps_data'                      # 后端API端点  
HEADERS = {'Content-Type': 'application/json'}      # 请求头  
  
# ----------------------- 本地监听端口配置 -----------------------------  
LOCAL_HOST = '192.168.3.10'                       # 本地监听地址  
LOCAL_PORT = 5000                                  # 本地监听端口  
LOCAL_ROUTE = '/notify_request_received'  # 本地监听路由  
  
# ----------------------- 串口配置 ------------------------------------  
PORT = '/dev/ttyUSB0'                               # 串口设备路径  
BAUDRATE = 9600                                     # 波特率  
BYTESIZE = 8                                        # 数据位  
PARITY = 'N'                                        # 校验位  
STOPBITS = 1                                        # 停止位  
TIMEOUT = 5                                         # 超时时间  
  
# ------------------------ Modbus寄存器地址 ----------------------------  
LATITUDE_REG = 150                                  # 纬度寄存器起始地址  
LONGITUDE_REG = 152                                 # 经度寄存器起始地址  
SPEED_REG = 156                                     # 速度寄存器起始地址  
NUM_REGISTERS = 2                                   # 读取的寄存器数量（每个浮点数占2个寄存器）  
  
# ----------------- 日志配置 ------------------------------------------  

time_str = datetime.datetime.now().strftime("%Y%m%d_%H%M%S")
logging.basicConfig(level=logging.INFO)  
logger = logging.getLogger(__name__)  
  
# 获取当前脚本所在目录的绝对路径  
script_dir = os.path.dirname(os.path.abspath(__file__))  
log_file_path = os.path.join(script_dir, f'./logs/arm_script_{time_str}.log') # 将日志文件放在脚本同级目录  
  
# 文件处理器  
file_handler = logging.FileHandler(log_file_path, encoding='utf-8')  
file_handler.setLevel(logging.INFO)  
formatter = logging.Formatter('%(asctime)s - %(levelname)s - %(message)s')  
file_handler.setFormatter(formatter)  
logger.addHandler(file_handler)  
  
# -----------------全局Modbus客户端实例，供每个Gunicorn worker使用--------------------  
  
_modbus_client_instance = None  
  
def get_modbus_client() -> ModbusSerialClient | None:  
  
    global _modbus_client_instance  
  
    if _modbus_client_instance is None or not _modbus_client_instance.connected:  
        if _modbus_client_instance is not None:  
            # 如果存在旧的客户端但已断开，尝试关闭它  
            try:  
                _modbus_client_instance.close()  
                logger.warning("[Modbus] 检测到连接断开或无效，尝试重新初始化Modbus客户端。")  
            except Exception as close_e:  
                logger.error(f"[Modbus] 关闭旧Modbus客户端时发生错误: {close_e}")  
  
        logger.info("[Modbus] Modbus客户端未连接或不存在，尝试创建并连接新的客户端实例。")  
        new_client = ModbusSerialClient(  
            port=PORT,  
            baudrate=BAUDRATE, # 修正为 BAUDRATE  
            bytesize=BYTESIZE,  
            parity=PARITY,  
            stopbits=STOPBITS,  
            timeout=TIMEOUT  
        )  
        try:  
            if not new_client.connect():  
                logger.error(f"[Modbus] 无法连接到Modbus串口 {PORT}。请检查设备是否连接或权限。")  
                _modbus_client_instance = None  # 连接失败时确保实例为 None  
                return None  
            else:  
                logger.info(f"[Modbus] 成功连接到Modbus RTU设备在 {PORT}")  
                _modbus_client_instance = new_client  
                return _modbus_client_instance  
        except Exception as e:  
            logger.error(f"[Modbus] 创建或连接Modbus客户端时发生异常: {e}")  
            _modbus_client_instance = None  # 连接失败时确保实例为 None  
            return None  
    return _modbus_client_instance  
  
#------------------Modbus客户端连接管理-------------------------------  
  
def read_gps_data_modbus() -> tuple[float | None, float | None, float | None]:  
    global _modbus_client_instance   
  
    # 获取当前 worker 对应的 Modbus 客户端实例  
    client = get_modbus_client()  
    if client is None:  
        logger.error("[Modbus] Modbus客户端不可用，无法读取GPS数据。")  
        return None, None, None  
  
    latitude = None         # 初始化经纬度，速度为None  
    longitude = None  
    speed = None  
  
    try:  
        # 读取纬度寄存器（寄存器地址，数量，slave默认为1）  
        result_lat_registers = client.read_holding_registers(address=LATITUDE_REG, count=NUM_REGISTERS, slave=1)  
        if result_lat_registers.isError():  
            logger.error(f"[Modbus] 读取纬度寄存器失败: {result_lat_registers}")  
            # 如果通信失败，可能指示连接已断开，强制下次重连  
            _modbus_client_instance = None # <-- 这里现在引用的是全局变量  
            return None, None, None  
  
        # 解析寄存器数据为浮点数  
        decoder_lat = BinaryPayloadDecoder.fromRegisters(  
            result_lat_registers.registers,  
            byteorder=Endian.BIG,  
            wordorder=Endian.BIG  
        )  
        latitude = decoder_lat.decode_32bit_float()     # 解析为32位浮点数  
  
        # 读取经度寄存器（寄存器地址，数量，slave默认为1）  
        result_lon_registers = client.read_holding_registers(address=LONGITUDE_REG, count=NUM_REGISTERS, slave=1)  
        if result_lon_registers.isError():  
            logger.error(f"[Modbus] 读取经度寄存器失败: {result_lon_registers}")  
            # 如果通信失败，可能指示连接已断开，强制下次重连  
            _modbus_client_instance = None 
        # 解析寄存器数据为浮点数  
        decoder_lon = BinaryPayloadDecoder.fromRegisters(  
            result_lon_registers.registers,  
            byteorder=Endian.BIG,  
            wordorder=Endian.BIG  
        )  
        longitude = decoder_lon.decode_32bit_float()    # 解析为32位浮点数  
  
        # 读取速度寄存器（寄存器地址，数量，slave默认为1）  
        result_spd_registers = client.read_holding_registers(address=SPEED_REG, count=NUM_REGISTERS, slave=1)  
        if result_spd_registers.isError():  
            logger.error(f"[Modbus] 读取速度寄存器失败: {result_spd_registers}")  
            # 如果通信失败，可能指示连接已断开，强制下次重连  
            _modbus_client_instance = None 
            return None, None, None  
        # 解析寄存器数据为浮点数  
        decoder_spd = BinaryPayloadDecoder.fromRegisters(  
            result_spd_registers.registers,  
            byteorder=Endian.BIG,  
            wordorder=Endian.BIG  
        )  
        speed = decoder_spd.decode_32bit_float()    # 解析为32位浮点数  
  
        if latitude is not None and longitude is not None and speed is not None:  
            logger.info(f"[Modbus GPS Data] Latitude = {latitude:.6f}, Longitude = {longitude:.6f}, Speed = {speed:.2f} km/h")  
        return latitude, longitude, speed * 1.852  # 设备的默认单位为节，这里转换为km/h  
  
    except Exception as e:  
        logger.error(f"[Modbus] 读取或解析GPS数据出错: {e}")  
        # 如果 Modbus 通信出现异常，强制将客户端标记为需要重新连接  
        _modbus_client_instance = None  
        return None, None, None  
  
#---------------------------------HTTP连接测试----------------------------------  
def test_http_connection(timeout: int = 5) -> tuple[bool, str]:  
  
    url = f"http://{BACKEND_HOST}:{BACKEND_PORT}"   # 后端 URL  
    logger.info(f"测试与后端的HTTP连接: {url}")  
  
    try:  
        response = requests.get(url, timeout=timeout, verify=True) # 检测连接  
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
    logger.info(f"尝试向 {url} 发送数据...")  
  
    try:  
        response = requests.post(url, json=data, headers=HEADERS, timeout=5)    # 发送POST请求  
        response.raise_for_status() # 如果响应状态码不是200-299，将引发HTTPError  
        logger.info(f"数据发送成功！Backend响应: {response.status_code} {response.text}")  
  
    except requests.exceptions.ConnectionError as e:  
        logger.error(f"发送数据失败，无法连接到后端 {url}: {e}")  
    except requests.exceptions.Timeout as e:  
        logger.error(f"发送数据失败，请求超时到后端 {url}: {e}")  
    except requests.exceptions.HTTPError as e:  
        logger.error(f"发送数据失败，后端返回HTTP错误 {e.response.status_code}: {e.response.text}")  
    except requests.exceptions.RequestException as e:  
        logger.error(f"发送数据时发生未知错误: {e}")  
    except Exception as e:  
        logger.error(f"发送数据时发生异常: {e}")  
  
  
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
        logger.warning("未获取到有效的GPS数据，跳过发送到后端。")  
  
  
#-----------------------------Flask路由定义--------------------------------------  
@app.route(LOCAL_ROUTE, methods=['POST', 'GET']) # 后端请求GPS数据的路由  
def listen_and_send():  
    logger.info("收到后端请求，将在后台线程中处理GPS数据...")  
    # 启动一个新线程来处理GPS数据读取和发送  
    thread = threading.Thread(target=process_gps_request_in_thread)  
    thread.start()  
    return jsonify({"status": "acknowledged", "message": "请求已接收，GPS数据正在后台处理并发送中。"}), 200  
  
  
#-----------------------------主程序入口--------------------------------------   
if __name__ == "__main__":    
    
    initial_client = get_modbus_client()    
    if initial_client is None:    
        logger.error(f"错误：无法连接到串口 {PORT}。请检查设备是否连接或权限。")    
        sys.exit(1) # 本地运行失败则直接退出    
    else:    
        logger.info("Modbus 客户端成功连接。")    
      
    # 循环检查后端连接，直到成功    
    while True:    
        if_http_isconnected, http_msg = test_http_connection()    
        if not if_http_isconnected:    
            logger.error(f"与后端的http连接失败，请检查网络设置或后端服务状态，\n错误信息：{http_msg}")    
            logger.info("准备进行重试...")    
            time.sleep(5)    
        else:    
            logger.info("与后端连接成功，本地服务将监听后端位置获取请求。")    
            break    
    
    logger.info(f"本地服务正在 {LOCAL_HOST}:{LOCAL_PORT} 上启动...")
    # 使用 Waitress 运行 Flask 应用
    serve(app, host=LOCAL_HOST, port=LOCAL_PORT)
      
    # 关闭Modbus客户端连接  
    if _modbus_client_instance and _modbus_client_instance.connected:    
        _modbus_client_instance.close()    
        logger.info("Modbus客户端已关闭。")

EOF
if [ $? -ne 0 ]; then
    echo "错误：无法创建Python脚本文件 '$PYTHON_SCRIPT_NAME'。"
    exit 1
fi
echo "Python脚本文件 '$PYTHON_SCRIPT_NAME' 创建成功。"

# 6. 运行Python脚本
echo "运行test_remote.py..."
python3 "$PYTHON_SCRIPT_NAME"
if [ $? -ne 0 ]; then
    echo "错误：Python脚本运行失败。"
fi

echo "脚本执行完毕或已被用户终止。"
