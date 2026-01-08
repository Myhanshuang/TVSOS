package com.tvsos.service.impl;

import com.tvsos.mapper.DriverMapper;
import com.tvsos.service.DriverService;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Service;

@Service
public class DriverServiceImpl implements DriverService {
    @Autowired
    private DriverMapper driverMapper;

    /**
     * 统计司机数量
     * @return
     */
    @Override
    public Integer count() {
        Integer sum = driverMapper.count();
        if (sum == null) {
            sum = 0;
        }
        return sum;
    }
}
