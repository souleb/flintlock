package plans

import (
	"context"
	"fmt"

	"github.com/weaveworks/flintlock/core/models"
	"github.com/weaveworks/flintlock/core/ports"
	portsctx "github.com/weaveworks/flintlock/core/ports/context"
	"github.com/weaveworks/flintlock/core/steps/microvm"
	"github.com/weaveworks/flintlock/core/steps/network"
	"github.com/weaveworks/flintlock/core/steps/runtime"
	"github.com/weaveworks/flintlock/pkg/defaults"
	"github.com/weaveworks/flintlock/pkg/log"
	"github.com/weaveworks/flintlock/pkg/planner"
)

type CreateOrUpdatePlanInput struct {
	StateDirectory string
	VM             *models.MicroVM
}

func MicroVMCreateOrUpdatePlan(input *CreateOrUpdatePlanInput) planner.Plan {
	return &microvmCreateOrUpdatePlan{
		vm:       input.VM,
		stateDir: input.StateDirectory,
		steps:    []planner.Procedure{},
	}
}

type microvmCreateOrUpdatePlan struct {
	vm       *models.MicroVM
	stateDir string

	steps []planner.Procedure
}

func (p *microvmCreateOrUpdatePlan) Name() string {
	return MicroVMCreateOrUpdatePlanName
}

func (p *microvmCreateOrUpdatePlan) Create(ctx context.Context) ([]planner.Procedure, error) {
	logger := log.GetLogger(ctx).WithField("component", "plans").WithField("planType", "microvmCreateOrUpdatePlan")
	logger.Tracef("creating CreateOrUpdate plan for microvm: %s", p.vm.ID)

	ports, ok := portsctx.GetPorts(ctx)
	if !ok {
		return nil, portsctx.ErrPortsMissing
	}

	if p.vm.Spec.DeletedAt != 0 {
		return []planner.Procedure{}, nil
	}

	p.ensureStatus()

	var err error
	switch {
	case p.vm.Spec.UpdatedAt != 0:
		err = p.update(ctx, ports)
	default:
		err = p.create(ctx, ports)
	}
	if err != nil {
		return nil, fmt.Errorf("error occurred generating plan: %w", err)
	}

	return p.steps, nil
}

func (p *microvmCreateOrUpdatePlan) Result() interface{} {
	return nil
}

func (p *microvmCreateOrUpdatePlan) create(ctx context.Context, ports *ports.Collection) error {
	if err := p.addStep(ctx, runtime.NewCreateDirectory(p.stateDir, defaults.DataDirPerm, ports.FileSystem)); err != nil {
		return fmt.Errorf("adding root dir step: %w", err)
	}

	// Images
	if err := p.addImageSteps(ctx, p.vm, ports.ImageService); err != nil {
		return fmt.Errorf("adding image steps: %w", err)
	}

	// Network interfaces
	if err := p.addNetworkSteps(ctx, p.vm, ports.NetworkService); err != nil {
		return fmt.Errorf("adding network steps: %w", err)
	}

	// MicroVM provider create
	if err := p.addStep(ctx, microvm.NewCreateStep(p.vm, ports.Provider)); err != nil {
		return fmt.Errorf("adding microvm create step: %w", err)
	}

	// MicroVM provider start
	if err := p.addStep(ctx, microvm.NewStartStep(p.vm, ports.Provider)); err != nil {
		return fmt.Errorf("adding microvm start step: %w", err)
	}

	return nil
}

func (p *microvmCreateOrUpdatePlan) update(ctx context.Context, ports *ports.Collection) error {
	return nil
}

func (p *microvmCreateOrUpdatePlan) addStep(ctx context.Context, step planner.Procedure) error {
	shouldDo, err := step.ShouldDo(ctx)
	if err != nil {
		return fmt.Errorf("checking if step %s should be included in plan: %w", step.Name(), err)
	}

	if shouldDo {
		p.steps = append(p.steps, step)
	}

	return nil
}

func (p *microvmCreateOrUpdatePlan) addImageSteps(ctx context.Context, vm *models.MicroVM, imageSvc ports.ImageService) error {
	for i := range vm.Spec.Volumes {
		vol := vm.Spec.Volumes[i]
		status, ok := vm.Status.Volumes[vol.ID]
		if !ok {
			status = &models.VolumeStatus{}
			vm.Status.Volumes[vol.ID] = status
		}
		if vol.Source.Container != nil {
			if err := p.addStep(ctx, runtime.NewVolumeMount(&vm.ID, &vol, status, imageSvc)); err != nil {
				return fmt.Errorf("adding  volume mount step: %w", err)
			}
		}
	}
	if string(vm.Spec.Kernel.Image) != "" {
		if err := p.addStep(ctx, runtime.NewKernelMount(vm, imageSvc)); err != nil {
			return fmt.Errorf("adding kernel mount step: %w", err)
		}
	}
	if vm.Spec.Initrd != nil {
		if err := p.addStep(ctx, runtime.NewInitrdMount(vm, imageSvc)); err != nil {
			return fmt.Errorf("adding initrd mount step: %w", err)
		}
	}

	return nil
}

func (p *microvmCreateOrUpdatePlan) addNetworkSteps(ctx context.Context, vm *models.MicroVM, networkSvc ports.NetworkService) error {
	for i := range vm.Spec.NetworkInterfaces {
		iface := vm.Spec.NetworkInterfaces[i]
		status, ok := vm.Status.NetworkInterfaces[iface.GuestDeviceName]
		if !ok {
			status = &models.NetworkInterfaceStatus{}
			vm.Status.NetworkInterfaces[iface.GuestDeviceName] = status
		}
		if err := p.addStep(ctx, network.NewNetworkInterface(&vm.ID, &iface, status, networkSvc)); err != nil {
			return fmt.Errorf("adding create network interface step: %w", err)
		}
	}

	return nil
}

func (p *microvmCreateOrUpdatePlan) ensureStatus() {
	if p.vm.Status.Volumes == nil {
		p.vm.Status.Volumes = models.VolumeStatuses{}
	}

	if p.vm.Status.NetworkInterfaces == nil {
		p.vm.Status.NetworkInterfaces = models.NetworkInterfaceStatuses{}
	}

	// I'll leave this condition here for safety. If (for some reason) it's
	// called on a vm that's not pending, leave the status as it is.
	if p.vm.Status.State == models.PendingState {
		p.vm.Status.State = models.CreatedState
	}
}
