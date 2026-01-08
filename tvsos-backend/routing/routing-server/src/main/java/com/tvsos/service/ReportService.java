package com.tvsos.service;

import vo.VehicleCategoryVO;

import java.util.List;

public interface ReportService {
    /**
     * 饼图 车辆类型 - 数量
     * @return
     */
    List<VehicleCategoryVO> reportVehicleCategory();

    /**
     * 柱状图 poi类型 - 数量
     * @return
     */
    List<List> reportPoiTybe();

    /**
     * 统计 车上的货物总量
     * @return
     */
    Double reportCargoSize();

    /**
     * 统计 poi 数量
     * @return
     */
    Integer reportPoiSum();

    /**
     * 统计车辆数量
     * @return
     */
    Integer reportVehicleSum();

    /**
     * 统计司机数量
     * @return
     */
    Integer reportDriverSum();

}
