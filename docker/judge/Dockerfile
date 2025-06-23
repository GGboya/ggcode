# 代码评测专用镜像 - 使用本地Alpine镜像加速
FROM docker4cn/alpine:3.10-aliyun

# 更新软件源为阿里云镜像
RUN echo "http://mirrors.aliyun.com/alpine/v3.10/main/" > /etc/apk/repositories && \
    echo "http://mirrors.aliyun.com/alpine/v3.10/community/" >> /etc/apk/repositories

# 安装基础工具和编译器
RUN apk add --no-cache \
    # C++ 编译器
    g++ \
    gcc \
    # Python
    python3 \
    py3-pip \
    # Java (包含javac编译器)
    openjdk11 \
    openjdk11-jdk \
    openjdk11-jre \
    # 系统工具
    bash \
    coreutils \
    curl \
    wget \
    vim

# 手动安装最新版本的Go
RUN wget https://golang.google.cn/dl/go1.24.4.linux-amd64.tar.gz && \
    tar -C /usr/local -xzf go1.24.4.linux-amd64.tar.gz && \
    rm go1.24.4.linux-amd64.tar.gz

# 设置 Java 环境变量 (Alpine Linux路径)
ENV JAVA_HOME=/usr/lib/jvm/java-11-openjdk
ENV PATH=$PATH:$JAVA_HOME/bin

# 设置 Go 环境变量
ENV GOROOT=/usr/local/go
ENV GOPATH=/home/judge/go
ENV PATH=$PATH:$GOROOT/bin:$GOPATH/bin

# 创建评测用户 (非root，提高安全性)
RUN adduser -D -s /bin/bash judge
RUN mkdir -p /opt/judge && chown judge:judge /opt/judge
RUN mkdir -p /home/judge/go && chown judge:judge /home/judge/go

# 设置工作目录
WORKDIR /opt/judge

# 创建 Java 安全策略文件
RUN echo 'grant {' > /opt/java.policy && \
    echo '  permission java.io.FilePermission "/opt/judge/-", "read,write,delete";' >> /opt/java.policy && \
    echo '  permission java.io.FilePermission "/tmp/-", "read,write,delete";' >> /opt/java.policy && \
    echo '  permission java.lang.RuntimePermission "exitVM";' >> /opt/java.policy && \
    echo '  permission java.util.PropertyPermission "*", "read";' >> /opt/java.policy && \
    echo '};' >> /opt/java.policy

# 复制评测脚本
COPY judge-script.sh /opt/judge/
RUN chmod +x /opt/judge/judge-script.sh

# 切换到评测用户
USER judge

# 验证环境
RUN g++ --version && \
    python3 --version && \
    java -version && \
    javac -version && \
    go version

CMD ["/bin/bash"] 