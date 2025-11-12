package com.tvsos.mapper;

import entity.Category;
import org.apache.ibatis.annotations.Mapper;
import java.util.Optional;

@Mapper
public interface CategoryMapper {
    /**
     * 根据ID查询 (用于查询车辆的装载能力)
     */
    Optional<Category> findById(Long categoryId);

    /**
     * 根据货物ID查询其规格
     */
    Optional<Category> findByCargoId(Long cargoId);
}