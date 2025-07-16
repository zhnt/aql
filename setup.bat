@echo off
chcp 65001 > nul
echo ================================================
echo 格力空调集中控制器监控程序 - 环境设置
echo ================================================
echo.

:: 检查Java环境
echo 检查Java环境...
java -version > nul 2>&1
if %errorlevel% neq 0 (
    echo [错误] 未找到Java环境！
    echo 请先安装JDK 8或更高版本
    echo 下载地址: https://www.oracle.com/java/technologies/downloads/
    pause
    exit /b 1
)
echo [✓] Java环境正常

:: 检查Maven环境
echo 检查Maven环境...
mvn -version > nul 2>&1
if %errorlevel% neq 0 (
    echo [错误] 未找到Maven环境！
    echo 请先安装Maven 3.6或更高版本
    echo 下载地址: https://maven.apache.org/download.cgi
    pause
    exit /b 1
)
echo [✓] Maven环境正常

:: 创建Maven项目结构
echo 创建项目目录结构...
if not exist "src\main\java" (
    mkdir "src\main\java"
)
if not exist "src\test\java" (
    mkdir "src\test\java"
)
if not exist "target" (
    mkdir "target"
)
echo [✓] 项目目录结构创建完成

:: 移动Java文件到正确位置
echo 组织项目文件...
if exist "GreeAirConditionerMonitor.java" (
    copy "GreeAirConditionerMonitor.java" "src\main\java\" > nul
    echo [✓] 主程序文件已移动
)
if exist "ModbusTest.java" (
    copy "ModbusTest.java" "src\main\java\" > nul
    echo [✓] 测试程序文件已移动
)
if exist "Demo.java" (
    copy "Demo.java" "src\main\java\" > nul
    echo [✓] 演示程序文件已移动
)

:: 下载依赖
echo 下载项目依赖...
mvn dependency:resolve
if %errorlevel% neq 0 (
    echo [错误] 依赖下载失败！
    echo 请检查网络连接和Maven配置
    pause
    exit /b 1
)
echo [✓] 依赖下载完成

:: 复制依赖到target目录
echo 复制依赖文件...
mvn dependency:copy-dependencies -DoutputDirectory=target/dependency
if %errorlevel% neq 0 (
    echo [警告] 依赖复制失败，程序可能无法正常运行
)

:: 编译项目
echo 编译项目...
mvn clean compile
if %errorlevel% neq 0 (
    echo [错误] 编译失败！
    pause
    exit /b 1
)
echo [✓] 编译成功

:: 创建可执行JAR
echo 创建可执行JAR文件...
mvn package
if %errorlevel% neq 0 (
    echo [警告] JAR文件创建失败
)

echo.
echo ================================================
echo 环境设置完成！
echo ================================================
echo.
echo 可用的运行命令：
echo 1. 运行主程序：         run.bat
echo 2. 运行通讯测试：       test.bat
echo 3. 运行演示程序：       demo.bat
echo 4. 直接运行JAR：        java -jar target/air-conditioner-monitor-1.0.0.jar
echo.
echo 使用说明：
echo - 确保已连接RS485转USB适配器
echo - 检查集中控制器电源和连接
echo - 确认网关地址配置正确（默认为1）
echo.
pause 