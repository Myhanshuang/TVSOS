package com.tvsos.mapper;

import entity.Cargo;
import org.apache.ibatis.annotations.Mapper;
import java.util.List;

@Mapper
public interface CargoMapper {
    /**
     * 根据货物类型查询
     */
    List<Cargo> findByType(Integer cargoType);

    /**
     * [新增] 获取所有货物列表
     */
    List<Cargo> list();
}