import json
import random
import os

def map_poi_type(original_type):
    """映射原始type到供应链层级1-10"""
    if 1 <= original_type <= 3:
        return original_type
    elif 4 <= original_type <= 9:
        return 4  # 工厂
    elif 10 <= original_type <= 15:
        return 5  # 汽修厂
    elif 16 <= original_type <= 18:
        return 6  # 物流园
    elif original_type == 19:
        return 7  # 火车站
    elif original_type == 20:
        return 8  # 机场
    elif 21 <= original_type <= 22:
        return 9  # 购物中心
    elif 23 <= original_type <= 25:
        return 10 # 家具建材市场
    return None

def process_poi_data(input_file, output_file):
    # --- 获取脚本所在目录 ---
    script_dir = os.path.dirname(os.path.abspath(__file__))
    
    # --- 构建输入文件的完整路径 ---
    input_path = os.path.join(script_dir, input_file)
    with open(input_path, 'r', encoding='utf-8') as f:
        data = json.load(f)
    
    pois = data.get("poi", [])
    
    # --- 按映射后的层级进行分组 ---
    groups = {}
    for poi in pois:
        mapped_type = map_poi_type(poi['type'])
        if mapped_type not in groups:
            groups[mapped_type] = []
        groups[mapped_type].append(poi['id'])

    # --- 为每个 POI 添加 prev 和 next 字段 ---
    for poi in pois:
        current_mapped_type = map_poi_type(poi['type'])
        
        # 处理上游 prev (层级 - 1)
        prev_type = current_mapped_type - 1
        if prev_type in groups and groups[prev_type]:
            poi['prev'] = random.choice(groups[prev_type])
        else:
            poi['prev'] = None
            
        # 处理下游 next (层级 + 1)
        next_type = current_mapped_type + 1
        if next_type in groups and groups[next_type]:
            poi['next'] = random.choice(groups[next_type])
        else:
            poi['next'] = None

    # --- 构建输出文件的完整路径并保存 ---
    output_path = os.path.join(script_dir, output_file)
    with open(output_path, 'w', encoding='utf-8') as f:
        json.dump(data, f, ensure_ascii=False, indent=4)
    
    print(f"处理完成！结果已保存至: {output_path}")

if __name__ == "__main__":
    process_poi_data('mockpoi.json', 'mockpoi_with_chain.json')