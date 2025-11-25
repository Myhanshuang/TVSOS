package com.tvsos.mapper;

import entity.Cargo;
import org.apache.ibatis.annotations.Mapper;
import org.apache.ibatis.annotations.Select;

import java.util.List;

@Mapper
public interface CargoMapper {

    @Select("select * from cargo")
    List<Cargo> findAll();

    @Select("select * from cargo where id = #{cargoId}")
    Cargo getById(Long cargoId);
}
