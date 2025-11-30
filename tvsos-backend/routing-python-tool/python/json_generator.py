import tkinter as tk
from tkinter import messagebox, filedialog, ttk
import json
import os


class JSONGeneratorApp:
    def __init__(self, master):
        self.master = master
        master.title("数据JSON生成器")
        master.geometry("800x700")

        # Updated to include 'id' field for all tables that have one
        self.db_schema = {
            "assign": ["id", "vehicle_id", "driver_id", "transport_order_id"],
            "cargo": ["id", "name", "type"],
            "category": ["id", "weight", "brand", "length", "width", "height", "capacity", "type", "scope"],
            "driver": ["id", "name", "status", "type"],
            "order_detail": ["transport_order_id", "cargo_id", "quantity"],
            "order_route": ["id", "transport_order_id", "route_id", "sequence", "status"],
            "poi": ["id", "name", "lon", "lat", "type", "status"],
            "route": ["id", "begin_poi_id", "end_poi_id", "distance", "type"],
            "transport_order": ["id", "order_number", "begin_poi_id", "end_poi_id", "est_begin_time", "est_end_time",
                                "act_begin_time", "act_end_time", "create_time", "status"],
            "vehicle": ["id", "license", "status", "lon", "lat", "speed", "category_id"]
        }

        self.notebook = ttk.Notebook(master)
        self.notebook.pack(expand=True, fill="both", padx=10, pady=10)

        self.tabs = {}
        for table_name in self.db_schema.keys():
            self.create_table_tab(table_name)

        self.generate_button = tk.Button(master, text="生成 JSON 文件", command=self.generate_json)
        self.generate_button.pack(pady=10)

    def create_table_tab(self, table_name):
        frame = ttk.Frame(self.notebook, padding=10)
        self.notebook.add(frame, text=table_name)

        canvas = tk.Canvas(frame)
        canvas.pack(side="left", fill="both", expand=True)

        scrollbar = ttk.Scrollbar(frame, orient="vertical", command=canvas.yview)
        scrollbar.pack(side="right", fill="y")
        canvas.configure(yscrollcommand=scrollbar.set)

        record_frame = ttk.Frame(canvas)
        canvas.create_window((0, 0), window=record_frame, anchor="nw")

        self.tabs[table_name] = {'frame': record_frame, 'records': [], 'canvas': canvas}

        add_button = tk.Button(frame, text=f"添加新{table_name}记录", command=lambda: self.add_record(table_name))
        add_button.pack(pady=5)

        self.add_record(table_name)  # Add one record by default

    def update_scrollregion(self, table_name):
        """Manually updates the canvas scroll region."""
        canvas = self.tabs[table_name]['canvas']
        canvas.update_idletasks()  # Ensure all widgets are rendered
        canvas.configure(scrollregion=canvas.bbox("all"))

    def add_record(self, table_name, initial_data=None):
        record_frame = self.tabs[table_name]['frame']

        new_record_row = tk.LabelFrame(record_frame, text=f"记录 {len(self.tabs[table_name]['records']) + 1}", padx=5,
                                       pady=5)
        new_record_row.pack(fill="x", padx=5, pady=5)

        entries = {}
        row_idx = 0
        for col_name in self.db_schema[table_name]:
            label = tk.Label(new_record_row, text=f"{col_name}:")
            label.grid(row=row_idx, column=0, sticky="w", padx=2, pady=2)

            entry = tk.Entry(new_record_row)
            if initial_data and col_name in initial_data:
                entry.insert(0, str(initial_data[col_name]))
            entry.grid(row=row_idx, column=1, sticky="ew", padx=2, pady=2)
            entries[col_name] = entry
            row_idx += 1

        remove_button = tk.Button(new_record_row, text="删除", fg="red",
                                  command=lambda: self.remove_record(table_name, new_record_row))
        remove_button.grid(row=0, column=2, sticky="ne", padx=5)

        new_record_row.grid_columnconfigure(1, weight=1)

        self.tabs[table_name]['records'].append({'frame': new_record_row, 'entries': entries})
        self.update_scrollregion(table_name)

    def remove_record(self, table_name, record_frame_to_remove):
        record_frame_to_remove.pack_forget()
        record_frame_to_remove.destroy()

        self.tabs[table_name]['records'] = [rec for rec in self.tabs[table_name]['records'] if
                                            rec['frame'] != record_frame_to_remove]

        for i, rec in enumerate(self.tabs[table_name]['records']):
            rec['frame'].config(text=f"记录 {i + 1}")

        self.update_scrollregion(table_name)

    def generate_json(self):
        output_data = {}
        for table_name, data in self.tabs.items():
            records = []
            for record_data in data['records']:
                record = {}
                is_record_empty = True
                for col_name, entry in record_data['entries'].items():
                    value = entry.get().strip()
                    if value:
                        is_record_empty = False
                        try:
                            if '.' in value:
                                record[col_name] = float(value)
                            else:
                                record[col_name] = int(value)
                        except ValueError:
                            record[col_name] = value
                if not is_record_empty:
                    records.append(record)

            if records:
                output_data[table_name] = records

        if not output_data:
            messagebox.showwarning("警告", "没有输入任何数据，无法生成 JSON 文件。")
            return

        file_path = filedialog.asksaveasfilename(
            defaultextension=".json",
            filetypes=[("JSON files", "*.json")],
            title="保存生成的 JSON 文件"
        )

        if file_path:
            try:
                with open(file_path, 'w', encoding='utf-8') as f:
                    json.dump(output_data, f, indent=4, ensure_ascii=False)
                messagebox.showinfo("成功", f"JSON 文件已成功生成并保存到：\n{os.path.basename(file_path)}")
            except Exception as e:
                messagebox.showerror("错误", f"保存文件时出错：\n{e}")


if __name__ == '__main__':
    root = tk.Tk()
    app = JSONGeneratorApp(root)
    root.mainloop()