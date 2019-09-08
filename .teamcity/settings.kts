package citruspi_froyg.buildTypes

import jetbrains.buildServer.configs.kotlin.v2018_2.*
import jetbrains.buildServer.configs.kotlin.v2018_2.buildSteps.ScriptBuildStep
import jetbrains.buildServer.configs.kotlin.v2018_2.buildSteps.script
import jetbrains.buildServer.configs.kotlin.v2018_2.triggers.vcs

object citruspi_froyg_build : BuildType({
    id("build")
    name = "build"

    vcs {
        root(citruspi_Froyg_HttpsSrcAlpacaHausCitruspiFroygGitRefsHeadsMaster)
    }

    steps {
        script {
            name = "go_fmt"
            scriptContent = "go fmt"
            dockerImage = "citruspi/go_xc_image:1.12.5-stretch"
            dockerImagePlatform = ScriptBuildStep.ImagePlatform.Linux
        }

        script {
            name = "go_vet"
            scriptContent = "go vet -c 3"
            dockerImage = "citruspi/go_xc_image:1.12.5-stretch"
            dockerImagePlatform = ScriptBuildStep.ImagePlatform.Linux
        }

    }

    triggers {
        vcs {
        }
    }
})
