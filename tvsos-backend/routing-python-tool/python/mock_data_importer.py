import tkinter as tk
from tkinter import filedialog, messagebox, scrolledtext
import json
import os
import re  # 导入正则表达式库用于名称转换
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

# --- 新增：驼峰命名转下划线命名的辅助函数 ---
def camel_to_snake(name):
    """
    将驼峰命名（camelCase）字符串转换为下划线命名（snake_case）。
    例如：'beginPoiId' -> 'begin_poi_id'
    """
    s1 = re.sub('(.)([A-Z][a-z]+)', r'\1_\2', name)
    return re.sub('([a-z0-9])([A-Z])', r'\1_\2', s1).lower()


class DatabaseImporterApp:
    def __init__(self, master):
        self.master = master
        master.title("通用数据库导入工具")
        master.geometry("600x600")

        self.db_config = {}
        self.json_data = None
        # self.config_file = "db_config.json"

        self.config_file = get_parent_dir_path("db_config.json")

        self.create_widgets()
        self.load_config()

    def create_widgets(self):
        # ... [GUI部分代码保持不变] ...
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
        self.json_path_label = tk.Label(action_frame, text="未选择 JSON 文件")
        self.json_path_label.pack(side="left", fill="x", expand=True, padx=(0, 5))
        self.select_file_button = tk.Button(action_frame, text="选择 JSON 文件", command=self.select_json_file)
        self.select_file_button.pack(side="left")
        self.import_button = tk.Button(action_frame, text="开始导入", command=self.start_import)
        self.import_button.pack(side="right")
        self.log_text = scrolledtext.ScrolledText(self.master, state='disabled', height=15)
        self.log_text.pack(fill="both", expand=True, padx=10, pady=5)

    def log(self, message):
        self.log_text.configure(state='normal')
        self.log_text.insert(tk.END, message + "\n")
        self.log_text.see(tk.END)
        self.log_text.configure(state='disabled')

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
                "host": self.entries["主机:"].get(), "port": int(self.entries["端口:"].get()),
                "user": self.entries["用户:"].get(), "password": self.entries["密码:"].get(),
                "database": self.entries["数据库名:"].get()
            }
            conn = mysql.connector.connect(**db_config)
            return conn
        except Exception as e:
            self.log(f"数据库连接失败: {e}")
            messagebox.showerror("连接失败", f"无法连接到数据库: {e}")
            return None

    def select_json_file(self):
        file_path = filedialog.askopenfilename(title="选择要导入的 JSON 文件",
                                               filetypes=(("JSON Files", "*.json"), ("All Files", "*.*")))
        if file_path:
            self.json_path_label.config(text=f"已选择: {os.path.basename(file_path)}")
            try:
                with open(file_path, 'r', encoding='utf-8') as f:
                    self.json_data = json.load(f)
                self.log(f"成功加载 JSON 文件: {file_path}")
            except Exception as e:
                self.json_data = None
                self.log(f"错误: 无法读取 JSON 文件 - {e}")
                messagebox.showerror("错误", f"无法读取 JSON 文件: {e}")

    def start_import(self):
        self.log_text.configure(state='normal')
        self.log_text.delete('1.0', tk.END)
        self.log_text.configure(state='disabled')
        if not self.json_data:
            self.log("错误: 未选择或加载 JSON 文件。")
            messagebox.showerror("错误", "请先选择并加载一个 JSON 文件。")
            return
        conn = self.get_db_connection()
        if not conn: return
        self.log("正在执行数据导入...")
        try:
            self.import_data(conn)
            self.log("数据导入操作已完成。")
            messagebox.showinfo("完成", "数据导入操作已完成。请查看日志。")
        except Exception as e:
            self.log(f"发生未知错误: {e}")
            messagebox.showerror("错误", f"发生未知错误: {e}")
        finally:
            if conn.is_connected():
                conn.close()
                self.log("数据库连接已关闭。")

    def import_data(self, conn):
        table_order = [
            'poi', 'category', 'cargo', 'driver', 'vehicle', 'route',
            'transport_order', 'assign', 'order_detail', 'order_route'
        ]

        cursor = conn.cursor()

        for table_name in table_order:
            if table_name not in self.json_data:
                self.log(f"跳过：在JSON文件中未找到表 '{table_name}'")
                continue

            records = self.json_data[table_name]
            if not isinstance(records, list) or not records:
                self.log(f"跳过：表 '{table_name}' 数据为空或格式不正确。")
                continue

            self.log(f"\n--- 正在处理表 '{table_name}' ---")

            # --- 核心改进：动态转换列名 ---
            # 1. 从JSON的第一条记录获取驼峰命名(camelCase)的键
            json_keys = list(records[0].keys())

            # 2. 自动转换为下划线命名(snake_case)
            db_columns = [camel_to_snake(key) for key in json_keys]

            # 3. 处理特殊的命名不一致情况
            if table_name == 'assign' and 'transport_id' in db_columns:
                db_columns[db_columns.index('transport_id')] = 'transport_order_id'
            if table_name == 'order_detail' and 'transport_id' in db_columns:
                db_columns[db_columns.index('transport_id')] = 'transport_order_id'
            # --- 改进结束 ---

            cursor.execute(f"SELECT id FROM `{table_name}`")
            existing_ids = {row[0] for row in cursor.fetchall()}

            records_to_insert = []
            records_to_update = []

            for record in records:
                record_id = record.get('id')
                # 按原始json_keys的顺序准备数据元组
                record_values = tuple(record.get(key) for key in json_keys)

                if record_id in existing_ids:
                    # 将ID放到元组最后，以匹配UPDATE语句的WHERE子句
                    update_tuple = tuple(record.get(key) for key in json_keys if key != 'id') + (record_id,)
                    records_to_update.append(update_tuple)
                else:
                    records_to_insert.append(record_values)

            try:
                if records_to_insert:
                    cols_for_insert = ", ".join([f"`{col}`" for col in db_columns])
                    placeholders = ", ".join(["%s"] * len(db_columns))
                    sql_insert = f"INSERT INTO `{table_name}` ({cols_for_insert}) VALUES ({placeholders})"
                    cursor.executemany(sql_insert, records_to_insert)
                    self.log(f"  - 批量插入 {len(records_to_insert)} 条新记录成功。")

                if records_to_update:
                    update_db_columns = [col for col in db_columns if col != 'id']
                    cols_for_update = ", ".join([f"`{col}`=%s" for col in update_db_columns])
                    sql_update = f"UPDATE `{table_name}` SET {cols_for_update} WHERE id = %s"
                    cursor.executemany(sql_update, records_to_update)
                    self.log(f"  - 批量更新 {len(records_to_update)} 条记录成功。")

                if records_to_insert or records_to_update:
                    conn.commit()

            except mysql.connector.Error as err:
                self.log(f"  - 错误: 处理表 '{table_name}' 失败 - {err}")
                conn.rollback()

        cursor.close()


if __name__ == '__main__':
    root = tk.Tk()
    app = DatabaseImporterApp(root)
    root.mainloop()