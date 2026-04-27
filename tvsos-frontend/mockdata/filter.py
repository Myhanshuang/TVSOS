import json

def filter_poi_data(input_file, output_file):
    """
    读取JSON文件，删除poi列表中prev和next均为null的项，并保存结果。
    
    :param input_file: 输入的JSON文件路径
    :param output_file: 输出的JSON文件路径
    """
    # 1. 读取JSON文件
    with open(input_file, 'r', encoding='utf-8') as f:
        data = json.load(f)
    
    # 2. 检查数据结构并过滤POI
    # 确保'poi'键存在且是一个列表
    if 'poi' in data and isinstance(data['poi'], list):
        # 使用列表推导式过滤：仅保留prev或next不为null的项
        # 即：删除 (prev is None AND next is None) 的项
        data['poi'] = [
            poi for poi in data['poi'] 
            if not (poi.get('prev') is None and poi.get('next') is None)
        ]
    else:
        print("警告: 数据中未找到 'poi' 列表，无法处理。")
        return

    # 3. 写入新的JSON文件
    with open(output_file, 'w', encoding='utf-8') as f:
        # ensure_ascii=False 保证中文正常显示，indent=2 用于格式化输出
        json.dump(data, f, ensure_ascii=False, indent=2)
    
    print(f"处理完成！原始数据包含 {len(data['poi']) + sum(1 for x in data.get('poi', []) if x.get('prev') is None and x.get('next') is None)} 个POI。")
    print(f"过滤后剩余 {len(data['poi'])} 个POI。")

# --- 使用示例 ---
# 请将 'input.json' 替换为你实际的文件名
# 请将 'output_filtered.json' 替换为你希望保存的文件名
if __name__ == "__main__":
    input_filename = 'mockpoi_with_chain.json' # 输入文件
    output_filename = 'filtered_poi_output.json' # 输出文件
    filter_poi_data(input_filename, output_filename)