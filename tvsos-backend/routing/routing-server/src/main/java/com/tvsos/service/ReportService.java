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
}
