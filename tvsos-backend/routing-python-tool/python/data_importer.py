import tkinter as tk
from tkinter import filedialog, messagebox, scrolledtext
import json
import os
import mysql.connector
from datetime import datetime


class DatabaseImporterApp:
    def __init__(self, master):
        self.master = master
        master.title("数据库导入工具")
        master.geometry("600x600")

        self.db_config = {}
        self.json_data = None
        self.config_file = "db_config.json"

        self.create_widgets()
        self.load_config()

    def create_widgets(self):
        # Frame for database settings
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

        # Action buttons for configuration
        config_buttons_frame = tk.Frame(settings_frame)
        config_buttons_frame.grid(row=len(labels), column=1, sticky="e", pady=5)

        save_button = tk.Button(config_buttons_frame, text="保存配置", command=self.save_config)
        save_button.pack(side="left", padx=(0, 5))

        clear_button = tk.Button(config_buttons_frame, text="清空配置", command=self.clear_config)
        clear_button.pack(side="left")

        settings_frame.grid_columnconfigure(1, weight=1)

        # Frame for file and actions
        action_frame = tk.Frame(self.master, padx=10, pady=5)
        action_frame.pack(fill="x", padx=10, pady=5)

        self.json_path_label = tk.Label(action_frame, text="未选择 JSON 文件")
        self.json_path_label.pack(side="left", fill="x", expand=True, padx=(0, 5))

        self.select_file_button = tk.Button(action_frame, text="选择 JSON 文件", command=self.select_json_file)
        self.select_file_button.pack(side="left")

        self.import_button = tk.Button(action_frame, text="开始导入", command=self.start_import)
        self.import_button.pack(side="right")

        # Frame for database actions
        db_action_frame = tk.Frame(self.master, padx=10, pady=5)
        db_action_frame.pack(fill="x", padx=10, pady=5)

        truncate_button = tk.Button(db_action_frame, text="一键清空数据库", fg="red", command=self.truncate_database)
        truncate_button.pack(side="right")

        # Log display
        self.log_text = scrolledtext.ScrolledText(self.master, state='disabled', height=10)
        self.log_text.pack(fill="both", expand=True, padx=10, pady=5)

    def log(self, message):
        self.log_text.configure(state='normal')
        self.log_text.insert(tk.END, message + "\n")
        self.log_text.see(tk.END)
        self.log_text.configure(state='disabled')

    def save_config(self):
        config_data = {key: entry.get() for key, entry in self.entries.items()}
        try:
            with open(self.config_file, 'w') as f:
                json.dump(config_data, f, indent=4)
            self.log(f"数据库配置已保存到 {self.config_file}")
            messagebox.showinfo("保存成功", f"数据库配置已保存到 {self.config_file}")
        except Exception as e:
            self.log(f"保存配置失败: {e}")
            messagebox.showerror("保存失败", f"保存配置失败: {e}")

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

    def clear_config(self):
        response = messagebox.askyesno("确认清空", "你确定要清空所有数据库配置信息吗？")
        if response:
            for entry in self.entries.values():
                entry.delete(0, tk.END)
            self.log("已清空数据库配置信息。")

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

    def truncate_database(self):
        response = messagebox.askyesno(
            "警告：清空数据库",
            "你确定要清空数据库 'routing' 中所有表的数据吗？\n此操作不可撤销！"
        )
        if not response:
            self.log("操作已取消。")
            return

        conn = self.get_db_connection()
        if not conn:
            return

        self.log("正在清空数据库 'routing' 中的所有数据...")
        cursor = conn.cursor()
        try:
            cursor.execute("SET FOREIGN_KEY_CHECKS = 0;")

            # Get all table names
            cursor.execute("SHOW TABLES")
            tables = [table[0] for table in cursor.fetchall()]

            for table_name in tables:
                cursor.execute(f"TRUNCATE TABLE `{table_name}`")
                self.log(f"  - 表 '{table_name}' 数据已清空。")

            cursor.execute("SET FOREIGN_KEY_CHECKS = 1;")
            conn.commit()
            self.log("数据库数据清空操作已完成。")
            messagebox.showinfo("完成", "数据库数据已全部清空。")
        except Exception as e:
            self.log(f"清空数据库失败: {e}")
            messagebox.showerror("错误", f"清空数据库失败: {e}")
        finally:
            conn.close()

    def select_json_file(self):
        file_path = filedialog.askopenfilename(
            title="选择要导入的 JSON 文件",
            filetypes=(("JSON Files", "*.json"), ("All Files", "*.*"))
        )
        if file_path:
            self.json_path_label.config(text=f"已选择: {file_path}")
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
        if not conn:
            return

        self.log("正在执行数据导入...")
        try:
            self.import_data(conn.cursor(), conn)
            self.log("数据导入操作已完成。")
            messagebox.showinfo("完成", "数据导入操作已完成。请查看日志。")
        except Exception as e:
            self.log(f"发生未知错误: {e}")
            messagebox.showerror("错误", f"发生未知错误: {e}")
        finally:
            conn.close()
            self.log("数据库连接已关闭。")

    def import_data(self, cursor, conn):
        for table_name, records in self.json_data.items():
            if not isinstance(records, list):
                self.log(f"错误: 表 '{table_name}' 的数据格式不正确，应为列表。")
                continue

            self.log(f"\n正在处理表 '{table_name}'...")
            for record in records:
                if 'id' in record:
                    # Check if record with this ID exists
                    check_sql = f"SELECT COUNT(*) FROM `{table_name}` WHERE id = %s"
                    cursor.execute(check_sql, (record['id'],))
                    exists = cursor.fetchone()[0] > 0

                    if exists:
                        self.update_record(cursor, conn, table_name, record)
                    else:
                        self.insert_record(cursor, conn, table_name, record)
                else:
                    self.insert_record(cursor, conn, table_name, record)

    def update_record(self, cursor, conn, table_name, record):
        try:
            record_id = record.pop('id')
            columns = [f"`{key}`=%s" for key in record]
            values = list(record.values())
            values.append(record_id)

            update_sql = f"UPDATE `{table_name}` SET {', '.join(columns)} WHERE id = %s"
            cursor.execute(update_sql, values)
            conn.commit()
            self.log(f"  - 更新记录 (ID: {record_id}) 成功。")
        except Exception as e:
            self.log(f"  - 错误: 更新表 '{table_name}' (ID: {record.get('id', '未知')}) 失败 - {e}")

    def insert_record(self, cursor, conn, table_name, record):
        try:
            columns = ", ".join([f"`{key}`" for key in record])
            placeholders = ", ".join(["%s"] * len(record))
            values = list(record.values())

            # Handle datetime/date objects if needed
            for i, val in enumerate(values):
                if isinstance(val, str) and ('-' in val or ':' in val) and ' ' not in val:
                    try:
                        # Try to parse as date
                        values[i] = datetime.strptime(val, '%Y-%m-%d').date()
                    except ValueError:
                        pass  # Not a date string, ignore

            insert_sql = f"INSERT INTO `{table_name}` ({columns}) VALUES ({placeholders})"
            cursor.execute(insert_sql, values)
            conn.commit()
            self.log(f"  - 插入新记录到表 '{table_name}' 成功。")
        except Exception as e:
            self.log(f"  - 错误: 插入新记录到表 '{table_name}' 失败 - {e}")


if __name__ == '__main__':
    root = tk.Tk()
    app = DatabaseImporterApp(root)
    root.mainloop()