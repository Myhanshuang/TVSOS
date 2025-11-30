import tkinter as tk
from tkinter import filedialog, messagebox, scrolledtext, ttk
import json
import random
import datetime
import string
import os
import mysql.connector
import threading
import sys
import os


def get_parent_dir_path(file_path):
    """
    获取一个文件在父目录中的完整路径。
    无论程序是从.py脚本运行还是从打包后的.exe运行，都能正确工作。
    """
    # 判断程序是否被打包
    if getattr(sys, 'frozen', False):
        # 如果是打包后的程序，基础路径是可执行文件的目录
        application_path = os.path.dirname(sys.executable)
    else:
        # 如果是直接运行的.py脚本，基础路径是脚本文件所在的目录
        application_path = os.path.dirname(os.path.abspath(__file__))

    # 向上回溯一级，找到父目录
    parent_directory = os.path.dirname(application_path)

    # 返回父目录中目标文件的完整路径
    return os.path.join(parent_directory, file_path)

# ==============================================================================
# 核心数据生成逻辑 (与之前基本相同，但将print改为调用log函数)
# ==============================================================================
class MockDataGenerator:
    def __init__(self, log_callback):
        self.mock_data = {}
        self.db_config = {}
        self.log = log_callback  # 使用回调函数来记录日志
        self.id_pools = {
            "poi": [], "driver": [], "vehicle": [], "cargo": [],
            "category": [], "route": [], "transport_order": []
        }

    def load_db_config(self, config_file="db_config.json"):
        if not os.path.exists(config_file):
            self.log(f"错误：数据库配置文件 '{config_file}' 未找到。")
            return False
        try:
            with open(config_file, 'r', encoding='utf-8') as f:
                config_raw = json.load(f)
                self.db_config = {
                    "host": config_raw.get("主机:"), "port": int(config_raw.get("端口:")),
                    "user": config_raw.get("用户:"), "password": config_raw.get("密码:"),
                    "database": config_raw.get("数据库名:")
                }
            self.log(f"成功从 '{config_file}' 加载数据库配置。")
            return True
        except Exception as e:
            self.log(f"错误：加载或解析数据库配置文件失败 - {e}")
            return False

    def read_pois_from_db(self):
        if not self.db_config:
            self.log("错误：数据库配置未加载，无法读取POI。")
            return False
        self.log("正在连接数据库以读取POI数据...")
        try:
            conn = mysql.connector.connect(**self.db_config)
            cursor = conn.cursor(dictionary=True)
            cursor.execute("SELECT id, name, lon, lat, type, status FROM poi")
            pois = cursor.fetchall()
            if not pois:
                self.log("警告：数据库中的 'poi' 表为空。")
                return False
            self.mock_data["poi"] = pois
            self.id_pools["poi"] = [poi['id'] for poi in pois]
            self.log(f"-> 成功从数据库读取了 {len(pois)} 条 [poi] 数据")
            return True
        except mysql.connector.Error as err:
            self.log(f"数据库错误：{err}")
            return False
        finally:
            if 'conn' in locals() and conn.is_connected():
                cursor.close()
                conn.close()
                self.log("数据库连接已关闭。")

    def generate_random_string(self, length=8):
        return ''.join(random.choice(string.ascii_letters + string.digits) for _ in range(length))

    def generate_license_plate(self):
        province = random.choice(["京", "沪", "粤", "苏", "浙", "川"])
        return f"{province}{random.choice(string.ascii_uppercase)}-{''.join(random.choices(string.ascii_uppercase + string.digits, k=5))}"

    def generate_datetime_iso(self, start_date=None):
        if start_date is None:
            start_date = datetime.datetime.now() - datetime.timedelta(days=30)
        return (start_date + datetime.timedelta(days=random.randint(0, 29),
                                                seconds=random.randint(0, 86399))).isoformat()

    def generate_drivers(self, count):
        drivers = []
        first_names = ["张", "李", "王", "赵", "刘", "陈"]
        last_names = ["伟", "芳", "娜", "强", "磊", "敏"]
        for i in range(1, count + 1):
            drivers.append({"id": i, "name": f"{random.choice(first_names)}{random.choice(last_names)}",
                            "type": random.choice([1, 2]), "status": random.choice([0, 1])})
            self.id_pools["driver"].append(i)
        self.mock_data["driver"] = drivers
        self.log(f"-> 生成了 {len(drivers)} 条 [driver] 数据")

    def generate_vehicles(self, count):
        vehicles = []
        now = datetime.datetime.now()
        for i in range(1, count + 1):
            vehicles.append({
                "id": i,
                "license": self.generate_license_plate(),
                "status": random.choice([0, 1, 2]),
                "lon": round(random.uniform(103.8, 104.2), 6),
                "lat": round(random.uniform(30.5, 30.8), 6),
                "speed": round(random.uniform(0, 80), 2),
                "createTime": (now - datetime.timedelta(days=random.randint(30, 365))).isoformat(),
                "updateTime": now.isoformat(),
                "categoryId": random.choice(self.id_pools["category"]) if self.id_pools["category"] else None
            })

            self.id_pools["vehicle"].append(i)
        self.mock_data["vehicle"] = vehicles
        self.log(f"-> 生成了 {len(vehicles)} 条 [vehicle] 数据")

    def generate_cargos_and_categories(self, cargo_count, cat_count):
        categories = []
        for i in range(1, cat_count + 1):
            categories.append({
                "id": i, "weight": round(random.uniform(10, 500), 2),
                "brand": random.choice(["品牌A", "品牌B", "品牌C"]),
                "length": round(random.uniform(1, 5), 2), "width": round(random.uniform(1, 3), 2),
                "height": round(random.uniform(0.5, 2.5), 2), "capacity": round(random.uniform(1, 10), 2),
                "type": random.randint(1, 3), "scope": random.randint(1, 2)
            })
            self.id_pools["category"].append(i)
        self.mock_data["category"] = categories
        self.log(f"-> 生成了 {len(categories)} 条 [category] 数据")
        cargos = []
        for i in range(1, cargo_count + 1):
            cargos.append(
                {"id": i, "name": random.choice(["电子产品", "生鲜水果", "建筑材料", "服装", "日用品"]) + f"-{i}",
                 "type": random.choice(self.id_pools["category"])})
            self.id_pools["cargo"].append(i)
        self.mock_data["cargo"] = cargos
        self.log(f"-> 生成了 {len(cargos)} 条 [cargo] 数据")

    def generate_routes(self, count):
        routes = []
        if not self.id_pools["poi"] or len(self.id_pools["poi"]) < 2:
            self.log("警告：POI数量不足(<2)，无法生成routes。")
            return
        for i in range(1, count + 1):
            begin, end = random.sample(self.id_pools["poi"], 2)
            routes.append({"id": i, "beginPoiId": begin, "endPoiId": end, "distance": round(random.uniform(5, 200), 2),
                           "type": random.randint(1, 4)})
            self.id_pools["route"].append(i)
        self.mock_data["route"] = routes
        self.log(f"-> 生成了 {len(routes)} 条 [route] 数据")

    def generate_orders_and_related(self, count):
        transport_orders = []
        if not self.id_pools["poi"] or len(self.id_pools["poi"]) < 2:
            self.log("警告：POI数量不足(<2)，无法生成transport_order。")
            return
        now = datetime.datetime.now()
        for i in range(1, count + 1):
            begin, end = random.sample(self.id_pools["poi"], 2)
            est_begin = now + datetime.timedelta(days=random.randint(1, 5))
            order = {
                "id": i, "orderNumber": f"ORD{now.year}{i:05d}", "beginPoiId": begin, "endPoiId": end,
                "estBeginTime": est_begin.strftime('%Y-%m-%d'),
                "estEndTime": (est_begin + datetime.timedelta(days=random.randint(1, 3))).strftime('%Y-%m-%d'),
                "actBeginTime": None, "actEndTime": None, "createTime": self.generate_datetime_iso(),
                "status": random.choice([0, 10, 20, 30, 40])
            }
            transport_orders.append(order)
            self.id_pools["transport_order"].append(i)
        self.mock_data["transport_order"] = transport_orders
        self.log(f"-> 生成了 {len(transport_orders)} 条 [transport_order] 数据")

        details, assigns, order_routes = [], [], []
        order_route_id = 1
        for order_id in self.id_pools["transport_order"]:
            if self.id_pools["vehicle"] and self.id_pools["driver"]:
                assigns.append({"id": order_id, "vehicleId": random.choice(self.id_pools["vehicle"]),
                                "driverId": random.choice(self.id_pools["driver"]), "transportId": order_id})
            if self.id_pools["cargo"]:
                for detail_idx, cargo_id in enumerate(random.sample(self.id_pools["cargo"], random.randint(1, 3))):
                    details.append(
                        {"id": (order_id - 1) * 3 + detail_idx + 1, "transportId": order_id, "cargoId": cargo_id,
                         "quantity": random.randint(10, 200)})
            if self.id_pools["route"]:
                for seq, route_id in enumerate(random.sample(self.id_pools["route"], random.randint(2, 4)), 1):
                    order_routes.append(
                        {"id": order_route_id, "transportOrderId": order_id, "routeId": route_id, "sequence": seq,
                         "status": random.choice([0, 1])})
                    order_route_id += 1

        self.mock_data["order_detail"] = details
        self.mock_data["assign"] = assigns
        self.mock_data["order_route"] = order_routes
        self.log(f"-> 生成了 {len(details)} 条 [order_detail] 数据")
        self.log(f"-> 生成了 {len(assigns)} 条 [assign] 数据")
        self.log(f"-> 生成了 {len(order_routes)} 条 [order_route] 数据")

    def generate_all(self, quantities,config_path):
        self.log("开始生成模拟数据...")
        if not self.load_db_config(config_path): return False # 使用传入的路径
        if not self.read_pois_from_db(): return False

        self.generate_drivers(quantities["drivers"])
        self.generate_cargos_and_categories(quantities["cargos"], quantities["categories"])
        self.generate_vehicles(quantities["vehicles"])
        self.generate_routes(quantities["routes"])
        self.generate_orders_and_related(quantities["orders"])
        self.log("所有模拟数据生成完毕。")
        return True

    def save_to_json(self, filename):
        try:
            with open(filename, 'w', encoding='utf-8') as f:
                json.dump(self.mock_data, f, indent=4, ensure_ascii=False)
            self.log(f"成功将数据写入到文件: {filename}")
        except Exception as e:
            self.log(f"错误：写入JSON文件失败 - {e}")


# ==============================================================================
# GUI 应用部分
# ==============================================================================
class MockDataGeneratorGUI:
    def __init__(self, master):
        self.master = master
        master.title("交互式模拟数据生成器")
        master.geometry("550x500")

        self.generator = MockDataGenerator(self.log)
        self.entries = {}

        self.db_config_path = get_parent_dir_path("db_config.json")
        self.create_widgets()
        self.check_db_config()

    def log(self, message):
        self.log_text.configure(state='normal')
        self.log_text.insert(tk.END, f"{datetime.datetime.now().strftime('%H:%M:%S')} - {message}\n")
        self.log_text.see(tk.END)
        self.log_text.configure(state='disabled')

    def check_db_config(self):
        if os.path.exists(self.db_config_path):
            self.db_status_label.config(text="数据库配置 (db_config.json): 已找到")
        else:
            self.db_status_label.config(text="数据库配置 (db_config.json): 未找到!")

    def create_widgets(self):
        # Frame for settings
        settings_frame = ttk.LabelFrame(self.master, text="生成数量配置", padding=(10, 5))
        settings_frame.pack(fill="x", padx=10, pady=5)

        # Labels and Entries
        fields = {
            "司机 (Drivers)": 10, "车辆 (Vehicles)": 8, "货物 (Cargos)": 15,
            "货物类别 (Categories)": 5, "路线 (Routes)": 30, "订单 (Orders)": 10
        }

        row = 0
        for text, default_val in fields.items():
            key = text.split(" ")[1][1:-1].lower()  # Extract key like 'drivers'

            label = ttk.Label(settings_frame, text=f"{text}:")
            label.grid(row=row, column=0, sticky="w", padx=5, pady=3)

            entry = ttk.Entry(settings_frame, width=10)
            entry.insert(0, str(default_val))
            entry.grid(row=row, column=1, sticky="w", padx=5, pady=3)
            self.entries[key] = entry
            row += 1

        settings_frame.grid_columnconfigure(0, weight=1)
        settings_frame.grid_columnconfigure(1, weight=1)

        # DB status and Generate button
        action_frame = ttk.Frame(self.master)
        action_frame.pack(fill="x", padx=10, pady=10)

        self.db_status_label = ttk.Label(action_frame, text="数据库配置: ...")
        self.db_status_label.pack(side="left", fill="x", expand=True)

        self.generate_button = ttk.Button(action_frame, text="生成数据并保存", command=self.start_generation_thread)
        self.generate_button.pack(side="right")

        # Log display
        log_frame = ttk.LabelFrame(self.master, text="日志输出", padding=(10, 5))
        log_frame.pack(fill="both", expand=True, padx=10, pady=5)

        self.log_text = scrolledtext.ScrolledText(log_frame, state='disabled', height=10)
        self.log_text.pack(fill="both", expand=True)

    def start_generation_thread(self):
        # Disable button to prevent multiple clicks
        self.generate_button.config(state="disabled")
        # Run the generation process in a new thread to avoid freezing the GUI
        thread = threading.Thread(target=self.run_generation)
        thread.daemon = True
        thread.start()

    def run_generation(self):
        self.log_text.configure(state='normal')
        self.log_text.delete('1.0', tk.END)
        self.log_text.configure(state='disabled')

        try:
            quantities = {key: int(entry.get()) for key, entry in self.entries.items()}
        except ValueError:
            self.log("错误: 请确保所有输入框中的值都是整数。")
            messagebox.showerror("输入错误", "请确保所有输入框中的值都是整数。")
            self.generate_button.config(state="normal")
            return

        self.log("=== 开始生成任务 ===")
        success = self.generator.generate_all(quantities, self.db_config_path)


        if success:
            file_path = filedialog.asksaveasfilename(
                title="保存生成的 JSON 文件",
                defaultextension=".json",
                filetypes=[("JSON files", "*.json")],
                initialfile="mock_data.json"
            )
            if file_path:
                self.generator.save_to_json(file_path)
            else:
                self.log("用户取消了保存操作。")
        else:
            self.log("因发生错误，生成任务中止。")

        self.log("=== 任务结束 ===")
        # Re-enable the button
        self.generate_button.config(state="normal")


if __name__ == '__main__':
    root = tk.Tk()
    app = MockDataGeneratorGUI(root)
    root.mainloop()