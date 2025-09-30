import tkinter as tk
from tkinter import filedialog, messagebox, scrolledtext
import json
import mysql.connector
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
class PoisJsonReaderApp:
    def __init__(self, master):
        self.master = master
        master.title("POI JSON 导入工具 (文件模式)")
        master.geometry("600x600")

        self.db_config = {}
        self.json_data = None
        # self.config_file = "db_config.json"

        # self.mapping_file = "poi_type_mapping.json"

        self.config_file = get_parent_dir_path("db_config.json")
        self.mapping_file = get_parent_dir_path("poi_type_mapping.json")
        self.type_cache = {}

        self.create_widgets()
        self.load_config()
        self.load_type_mapping()

    def create_widgets(self):
        settings_frame = tk.LabelFrame(self.master, text="数据库配置", padx=10, pady=10)
        settings_frame.pack(fill="x", padx=10, pady=5)

        labels = ["主机:", "端口:", "用户:", "密码:", "数据库名:"]
        self.entries = {}
        defaults = {"主机:": "localhost", "端口:": "3306", "用户:": "root", "密码:": "", "数据库名:": "routing"}
        for i, label_text in enumerate(labels):
            label = tk.Label(settings_frame, text=label_text)
            label.grid(row=i, column=0, sticky="w", pady=2)
            entry = tk.Entry(settings_frame)
            entry.insert(0, defaults[label_text])
            if label_text == "密码:":
                entry.config(show="*")
            entry.grid(row=i, column=1, sticky="ew", padx=5)
            self.entries[label_text] = entry

        settings_frame.grid_columnconfigure(1, weight=1)

        action_frame = tk.Frame(self.master, padx=10, pady=5)
        action_frame.pack(fill="x", padx=10, pady=5)

        self.json_path_label = tk.Label(action_frame, text="未选择 POI JSON 文件")
        self.json_path_label.pack(side="left", fill="x", expand=True, padx=(0, 5))

        select_button = tk.Button(action_frame, text="选择 JSON 文件", command=self.select_json_file)
        select_button.pack(side="left")

        import_button = tk.Button(action_frame, text="开始导入", command=self.start_import)
        import_button.pack(side="right")

        self.log_text = scrolledtext.ScrolledText(self.master, state='disabled', height=15)
        self.log_text.pack(fill="both", expand=True, padx=10, pady=5)

    def log(self, message):
        self.log_text.configure(state='normal')
        self.log_text.insert(tk.END, message + "\n")
        self.log_text.see(tk.END)
        self.log_text.configure(state='disabled')

    def load_type_mapping(self):
        if os.path.exists(self.mapping_file):
            try:
                with open(self.mapping_file, 'r', encoding='utf-8') as f:
                    self.type_cache = json.load(f)
                self.log(f"成功从 {self.mapping_file} 加载 {len(self.type_cache)} 条类型映射。")
            except Exception as e:
                self.log(f"警告: 加载类型映射文件失败 - {e}")
        else:
            self.log("未找到类型映射文件，将在发现新类型时自动创建。")

    def save_type_mapping(self):
        try:
            with open(self.mapping_file, 'w', encoding='utf-8') as f:
                json.dump(self.type_cache, f, indent=4, ensure_ascii=False)
            self.log(f"类型映射已成功保存到 {self.mapping_file}")
        except Exception as e:
            self.log(f"错误: 保存类型映射文件失败 - {e}")

    def load_config(self):
        if os.path.exists(self.config_file):
            try:
                with open(self.config_file, 'r') as f:
                    config_data = json.load(f)
                for key, value in config_data.items():
                    if key in self.entries:
                        self.entries[key].delete(0, tk.END)
                        self.entries[key].insert(0, value)
                self.log("已加载保存的数据库配置。")
            except Exception as e:
                self.log(f"加载配置失败: {e}")

    def get_db_connection(self):
        try:
            db_config = {
                "host": self.entries["主机:"].get(),
                "port": int(self.entries["端口:"].get()),
                "user": self.entries["用户:"].get(),
                "password": self.entries["密码:"].get(),
                "database": self.entries["数据库名:"].get()
            }
            conn = mysql.connector.connect(**db_config)
            return conn
        except Exception as e:
            self.log(f"数据库连接失败: {e}")
            messagebox.showerror("连接失败", f"无法连接到数据库: {e}")
            return None

    def select_json_file(self):
        file_path = filedialog.askopenfilename(
            title="选择 POI JSON 文件",
            filetypes=(("JSON Files", "*.json"), ("All Files", "*.*"))
        )
        if file_path:
            self.json_path_label.config(text=f"已选择: {os.path.basename(file_path)}")
            try:
                with open(file_path, 'r', encoding='utf-8') as f:
                    self.json_data = json.load(f)
                if not isinstance(self.json_data, list):
                    self.json_data = None
                    raise ValueError("JSON 文件根结构必须是列表 [...]")
                self.log(f"成功加载 JSON 文件: {file_path}")
            except Exception as e:
                self.json_data = None
                self.log(f"错误: 无法读取或解析 JSON 文件 - {e}")
                messagebox.showerror("错误", f"无法读取 JSON 文件: {e}")

    def start_import(self):
        self.log_text.configure(state='normal')
        self.log_text.delete('1.0', tk.END)
        self.log_text.configure(state='disabled')

        if not self.json_data:
            self.log("错误: 未选择或加载 JSON 文件。")
            messagebox.showerror("错误", "请先选择并加载一个有效的 POI JSON 文件。")
            return

        conn = self.get_db_connection()
        if not conn:
            return

        self.log("开始导入 POI 数据...")
        try:
            self.import_pois(conn)
            self.log("POI 数据导入操作已完成。")
            self.save_type_mapping()
            messagebox.showinfo("完成", "数据导入操作已完成。请查看日志。")
        except Exception as e:
            self.log(f"发生未知错误: {e}")
            messagebox.showerror("错误", f"发生未知错误: {e}")
        finally:
            if conn.is_connected():
                conn.close()
                self.log("数据库连接已关闭。")

    # --- 核心修改：此方法不再与数据库交互 ---
    def get_or_assign_poi_type_id(self, type_name):
        """
        从内存缓存中获取类型ID。如果类型不存在，则分配一个新的ID。
        """
        if type_name in self.type_cache:
            return self.type_cache[type_name]
        else:
            # 类型不存在，需要分配新ID
            # 查找当前缓存中的最大ID值
            if not self.type_cache:
                new_id = 1  # 如果缓存为空，从1开始
            else:
                max_id = max(self.type_cache.values())
                new_id = max_id + 1

            self.log(f"  - 发现新类型 '{type_name}'，已分配新 ID: {new_id}")
            self.type_cache[type_name] = new_id  # 更新缓存
            return new_id

    def import_pois(self, conn):
        cursor = conn.cursor()

        for record in self.json_data:
            if not all(k in record for k in ["id", "name", "lon", "lat", "type"]):
                self.log(f"  - 警告: 跳过格式不完整的记录: {record}")
                continue

            record_id = record['id']
            poi_type_name = record['type']

            try:
                # --- 修改：调用新的、不依赖数据库的函数 ---
                type_id = self.get_or_assign_poi_type_id(poi_type_name)

                poi_data = {
                    'id': record_id,
                    'name': record['name'],
                    'lon': record['lon'],
                    'lat': record['lat'],
                    'type': type_id,
                }

                status_val = record.get('status')
                if status_val is not None:
                    try:
                        poi_data['status'] = int(status_val)
                    except (ValueError, TypeError):
                        self.log(f"  - 警告: POI (ID: {record_id}) 的 status '{status_val}' "
                                 f"无法转换为整数，将忽略此字段。")

                cursor.execute("SELECT COUNT(*) FROM poi WHERE id = %s", (record_id,))
                exists = cursor.fetchone()[0] > 0

                if exists:
                    update_cols = [f"`{key}`=%s" for key in poi_data if key != 'id']
                    update_vals = [val for key, val in poi_data.items() if key != 'id']

                    if not update_cols:
                        self.log(f"  - POI (ID: {record_id}) 无需更新。")
                        continue

                    update_vals.append(record_id)
                    sql = f"UPDATE `poi` SET {', '.join(update_cols)} WHERE id = %s"
                    cursor.execute(sql, update_vals)
                    self.log(f"  - 更新 POI (ID: {record_id}) 成功。")
                else:
                    columns = ", ".join([f"`{k}`" for k in poi_data])
                    placeholders = ", ".join(["%s"] * len(poi_data))
                    values = list(poi_data.values())
                    sql = f"INSERT INTO `poi` ({columns}) VALUES ({placeholders})"
                    cursor.execute(sql, values)
                    self.log(f"  - 插入新 POI (ID: {record_id}) 成功。")

                conn.commit()

            except mysql.connector.Error as err:
                self.log(f"  - 错误: 处理 POI (ID: {record_id}) 失败 - {err}")
                conn.rollback()

        cursor.close()


if __name__ == '__main__':
    root = tk.Tk()
    app = PoisJsonReaderApp(root)
    root.mainloop()